package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/ent"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

var (
	dialectArg string
	dbVersion  string
	outDirFlag string
)

// dumpToolDir 临时 dump 程序的目录名(位于模块根目录下,跑完即删)。
// 使用普通名字而非 ./_ 前缀,保证 `go run ./...` 显式路径在所有 Go 版本可用。
const dumpToolDir = "gow-migrate-tmp"

// CmdMigrate migrate 命令:把服务的 ent schema 逆向为 SQL DDL。
var CmdMigrate = &cobra.Command{
	Use:   "migrate [service...]",
	Short: "Dump ent schema DDL into SQL files (ent schema -> SQL, reverse direction)",
	Long: `Dump the ent schema of one, several, or all services into SQL DDL.

For every service with an ent schema (app/<service>/service/internal/data/ent),
the table definitions of the generated ent/migrate package are rendered into a
CREATE TABLE script for the target dialect via ent's schema.DDL — no live
database connection is required.

The SQL file is written to migrations/schema.<dialect>.sql inside each service
directory; --out redirects all files into a single directory (files are then
named <service>.<dialect>.sql). Services without an ent schema are skipped
with a warning; missing ent codegen is filled in automatically (as gow ent
does). Useful as an Atlas baseline migration or for reviewing schema state.

Examples:
  gow migrate                  # every ent-based service
  gow migrate admin            # one service
  gow migrate admin --dialect postgres
  gow migrate --dialect sqlite --out ./dist/sql`,
	RunE:         RunDump,
	SilenceUsage: true,
}

func init() {
	CmdMigrate.Flags().StringVar(&dialectArg, "dialect", "mysql", "target SQL dialect: mysql, postgres, sqlite")
	CmdMigrate.Flags().StringVar(&dbVersion, "db-version", "", "target database version for dialect-specific DDL (e.g. 8, 5.7, 14)")
	CmdMigrate.Flags().StringVarP(&outDirFlag, "out", "o", "", "output directory for all SQL files (default: each service's migrations/)")
}

// RunDump 为 cobra 的 RunE 回调:解析服务并逐个导出 DDL。
func RunDump(cmd *cobra.Command, args []string) error {
	dialect, err := normalizeDialect(dialectArg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	inspector, err := pkg.NewModuleInspectorFromGo("")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	// dump 程序引用 entgo 的 DDL 规划器,先确保依赖图完整(与 run/build 一致)。
	if err = pkg.GoModTidy(cmd.Context(), inspector.Root); err != nil {
		return err
	}

	names, err := resolveServiceNames(inspector.Root, args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	failed := false
	for _, name := range names {
		outPath, err := dumpService(inspector, name, dialect)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: dump DDL for service '%s' failed: %s\033[m\n", name, err.Error())
			failed = true
			continue
		}
		_, _ = fmt.Fprintf(os.Stdout, "\033[32mSUCCESS: dumped DDL for '%s' (%s) -> %s\033[m\n", name, dialect, outPath)
	}
	if failed {
		return fmt.Errorf("one or more services failed to dump DDL")
	}
	return nil
}

// resolveServiceNames 解析目标服务:无参数时枚举全部含 ent schema 的服务,
// 否则逐个校验服务存在性(是否 ent 型在 dump 阶段判定)。
func resolveServiceNames(root string, args []string) ([]string, error) {
	if len(args) == 0 {
		names, err := pkg.ListServiceNames(root)
		if err != nil {
			return nil, err
		}
		entNames := make([]string, 0, len(names))
		for _, name := range names {
			if hasEntSchema(root, name) {
				entNames = append(entNames, name)
			}
		}
		if len(entNames) == 0 {
			return nil, fmt.Errorf("no ent-based services found under %s", filepath.Join(root, "app"))
		}
		return entNames, nil
	}

	names := make([]string, 0, len(args))
	for _, arg := range args {
		name := strings.TrimSpace(arg)
		if name == "" {
			return nil, fmt.Errorf("service name is required")
		}
		valid, err := pkg.IsValidServiceName(root, name)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, fmt.Errorf("service '%s' does not exist or is not valid (missing cmd/server or configs)", name)
		}
		names = append(names, name)
	}
	return names, nil
}

func hasEntSchema(root string, name string) bool {
	schemaDir := filepath.Join(root, "app", name, "service", "internal", "data", "ent", "schema")
	fi, err := os.Stat(schemaDir)
	return err == nil && fi.IsDir()
}

// dumpService 把单个服务的 ent schema 导出为 DDL 文件,返回输出路径。
func dumpService(inspector *pkg.ModuleInspector, name string, dialect string) (string, error) {
	svcDir := filepath.Join(inspector.Root, "app", name, "service")
	entRoot := filepath.Join(svcDir, "internal", "data", "ent")

	if !hasEntSchema(inspector.Root, name) {
		return "", fmt.Errorf("no ent schema under %s; service is not ent-based", filepath.Join(entRoot, "schema"))
	}

	// schema.DDL 消费 ent/migrate 包里的表定义;代码尚未生成时自动补齐。
	if _, err := os.Stat(filepath.Join(entRoot, "migrate", "schema.go")); err != nil {
		if err = ent.GenerateService(svcDir); err != nil {
			return "", fmt.Errorf("ent codegen required for migrate package: %w", err)
		}
	}

	outPath, err := outputSQLPath(inspector.Root, name, dialect)
	if err != nil {
		return "", err
	}

	importPath := strings.TrimSuffix(
		filepath.ToSlash(filepath.Join(inspector.ModPath, "app", name, "service", "internal", "data", "ent", "migrate")),
		"/",
	)
	program, err := renderDumpProgram(importPath)
	if err != nil {
		return "", err
	}

	tmpDir := filepath.Join(svcDir, dumpToolDir)
	if err = os.RemoveAll(tmpDir); err != nil {
		return "", err
	}
	if err = os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", err
	}
	programPath := filepath.Join(tmpDir, "main.go")
	if err = os.WriteFile(programPath, []byte(program), 0o644); err != nil {
		return "", err
	}

	// go run 以服务目录为工作目录:模块自服务目录向上解析,同时满足
	// internal 包的可见性规则(导入方必须在 <svc>/... 子树内)。
	g := pkg.NewGoCmd(svcDir)
	runErr := g.Run("run", "./"+dumpToolDir, outPath, dialect, dbVersion)

	// 成功即清理;失败保留现场便于排查,并在错误信息里给出路径。
	if runErr != nil {
		return "", fmt.Errorf("dump program failed (kept at %s): %w", programPath, runErr)
	}
	_ = os.RemoveAll(tmpDir)
	return outPath, nil
}

// outputSQLPath 决定 SQL 输出路径:默认各服务 migrations/schema.<dialect>.sql,
// --out 时集中到单目录,文件名带服务名避免互相覆盖。
func outputSQLPath(root string, name string, dialect string) (string, error) {
	outDir := filepath.Join(root, "app", name, "service", "migrations")
	fileName := "schema." + dialect + ".sql"
	if outDirFlag != "" {
		abs, err := filepath.Abs(outDirFlag)
		if err != nil {
			return "", err
		}
		outDir = abs
		fileName = name + "." + dialect + ".sql"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(outDir, fileName), nil
}

// normalizeDialect 归一化方言名(与 ent 的 dialect 常量对齐:
// mysql / postgres / sqlite3)。
func normalizeDialect(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "mysql":
		return "mysql", nil
	case "postgres", "postgresql", "pg":
		return "postgres", nil
	case "sqlite", "sqlite3":
		return "sqlite3", nil
	default:
		return "", fmt.Errorf("unsupported dialect %q (supported: mysql, postgres, sqlite)", v)
	}
}

// dumpProgramTmpl 临时 dump 程序模板。方言与版本经命令行参数传入而非内嵌,
// 避免把用户输入拼进 Go 源码。
var dumpProgramTmpl = template.Must(template.New("dump").Parse(`// Code generated by gow migrate; DO NOT EDIT.

package main

import (
	"context"
	"fmt"
	"os"

	schemadsl "entgo.io/ent/dialect/sql/schema"

	migrate "{{.ImportPath}}"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: dump <output.sql> <dialect> <db-version>")
		os.Exit(2)
	}
	out, dialect, version := os.Args[1], os.Args[2], os.Args[3]

	sql, err := schemadsl.DDL(context.Background(), schemadsl.DDLArgs{
		Dialect: dialect,
		Version: version,
		Tables:  migrate.Tables,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "plan ddl:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, []byte(sql), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write ddl:", err)
		os.Exit(1)
	}
}
`))

type dumpProgramArgs struct {
	ImportPath string
}

func renderDumpProgram(importPath string) (string, error) {
	var buf strings.Builder
	if err := dumpProgramTmpl.Execute(&buf, dumpProgramArgs{ImportPath: importPath}); err != nil {
		return "", err
	}
	return buf.String(), nil
}
