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
	dialectArg    string
	dbVersion     string
	outDirFlag    string
	versionedFlag bool
	devURLFlag    string
	toolFlag      string
	nameFlag      string
)

// dumpToolDir 临时 dump 程序的目录名(位于模块根目录下,跑完即删)。
// 使用普通名字而非 ./_ 前缀,保证 `go run ./...` 显式路径在所有 Go 版本可用。
const dumpToolDir = "gow-migrate-tmp"

// CmdMigrate migrate 命令:把服务的 ent schema 逆向为 SQL DDL。
var CmdMigrate = &cobra.Command{
	Use:   "migrate [service...]",
	Short: "Dump ent schema DDL or generate versioned migration files",
	Long: `Two modes:

DDL dump (default) — for every service with an ent schema, render the table
definitions of the generated ent/migrate package into a CREATE TABLE script
for the target dialect via ent's schema.DDL — no live database connection is
required. The SQL file is written to migrations/schema.<dialect>.sql inside
each service directory; --out redirects all files into a single directory.
Services without an ent schema are skipped with a warning; missing ent
codegen is filled in automatically (as gow ent does).

Versioned migrations (--versioned) — for every service with an ent schema,
diff the ent schema against a development database and write the changes as
versioned migration files into the service's migrations/ directory, in the
file format of the selected migration tool. The dev database URL is
required and is never written into the generated program (it is passed as a
command-line argument at runtime). This follows ent's documented versioned
migration flow (schema.WithDir + ModeReplay + WithFormatter).

Examples:
  gow migrate                          # DDL dump, every ent-based service
  gow migrate admin --dialect postgres # DDL dump, one service
  gow migrate --versioned --url mysql://root:pass@localhost:3306/test
  gow migrate --versioned --tool golang-migrate --url "postgres://user:pass@localhost:5432/test?sslmode=disable"
  gow migrate admin --versioned --url mysql://root:pass@localhost:3306/test`,
	RunE:         RunDump,
	SilenceUsage: true,
}

func init() {
	CmdMigrate.Flags().StringVar(&dialectArg, "dialect", "mysql", "target SQL dialect: mysql, postgres, sqlite")
	CmdMigrate.Flags().StringVar(&dbVersion, "db-version", "", "target database version for dialect-specific DDL (e.g. 8, 5.7, 14)")
	CmdMigrate.Flags().StringVarP(&outDirFlag, "out", "o", "", "output directory for all SQL files (default: each service's migrations/)")
	CmdMigrate.Flags().BoolVar(&versionedFlag, "versioned", false, "generate versioned migration files by diffing the ent schema against a dev database")
	CmdMigrate.Flags().StringVar(&devURLFlag, "url", "", "dev database URL for --versioned (e.g. mysql://user:pass@host:3306/db)")
	CmdMigrate.Flags().StringVar(&toolFlag, "tool", "atlas", "migration file format for --versioned: atlas, golang-migrate, goose, db-mate, flyway, liquibase")
	CmdMigrate.Flags().StringVar(&nameFlag, "name", "changes", "migration name for --versioned (sanitized to [a-z0-9_-])")
}

// RunDump 为 cobra 的 RunE 回调:解析服务并按模式分派。
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

	if versionedFlag {
		return runVersionedAll(cmd, inspector, names, dialect)
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

// ==================== 版本化迁移(--versioned) ====================

// migTool 描述一种迁移文件格式(目录构造器 + 格式化器)。
type migTool struct {
	Name        string
	DirCall     string
	FormatCall  string
	UsesSQLTool bool
}

var migTools = map[string]migTool{
	"atlas": {
		Name:       "atlas",
		DirCall:    "atlas.NewLocalDir",
		FormatCall: "atlas.DefaultFormatter",
	},
	"golang-migrate": {
		Name:        "golang-migrate",
		DirCall:     "sqltool.NewGolangMigrateDir",
		FormatCall:  "sqltool.GolangMigrateFormatter",
		UsesSQLTool: true,
	},
	"goose": {
		Name:        "goose",
		DirCall:     "sqltool.NewGooseDir",
		FormatCall:  "sqltool.GooseFormatter",
		UsesSQLTool: true,
	},
	"db-mate": {
		Name:        "db-mate",
		DirCall:     "sqltool.NewDBMateDir",
		FormatCall:  "sqltool.DBMateFormatter",
		UsesSQLTool: true,
	},
	"flyway": {
		Name:        "flyway",
		DirCall:     "sqltool.NewFlywayDir",
		FormatCall:  "sqltool.FlywayFormatter",
		UsesSQLTool: true,
	},
	"liquibase": {
		Name:        "liquibase",
		DirCall:     "sqltool.NewLiquibaseDir",
		FormatCall:  "sqltool.LiquibaseFormatter",
		UsesSQLTool: true,
	},
}

// entDialectConst 方言名 → ent dialect 常量名与连接驱动 blank import。
var entDialectConst = map[string]struct{ Const, Driver string }{
	"mysql":    {Const: "dialect.MySQL", Driver: `_ "github.com/go-sql-driver/mysql"`},
	"postgres": {Const: "dialect.PostgreSQL", Driver: `_ "github.com/lib/pq"`},
	"sqlite3":  {Const: "dialect.SQLite", Driver: `_ "github.com/mattn/go-sqlite3"`},
}

// sanitizeMigName 清洗迁移名:仅保留小写字母数字与 -_，其余替换为 -，
// 空/全非法时返回 "changes"。
func sanitizeMigName(v string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(v)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), "-_")
	if s == "" {
		return "changes"
	}
	return s
}

// runVersionedAll 版本化模式入口:校验参数后逐服务生成版本化迁移文件。
func runVersionedAll(cmd *cobra.Command, inspector *pkg.ModuleInspector, names []string, dialect string) error {
	if strings.TrimSpace(devURLFlag) == "" {
		return fmt.Errorf("--versioned requires a dev database URL via --url (e.g. mysql://user:pass@host:3306/db)")
	}
	if outDirFlag != "" {
		return fmt.Errorf("--out applies to the DDL dump mode only; versioned migrations always go to each service's migrations/ directory")
	}
	tool, ok := migTools[strings.ToLower(strings.TrimSpace(toolFlag))]
	if !ok {
		return fmt.Errorf("unsupported --tool %q (supported: atlas, golang-migrate, goose, db-mate, flyway, liquibase)", toolFlag)
	}
	dc, ok := entDialectConst[dialect]
	if !ok {
		return fmt.Errorf("unsupported dialect %q for versioned migrations", dialect)
	}

	failed := false
	for _, name := range names {
		dir, err := versionedService(cmd, inspector, name, dialect, tool, dc)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: versioned migration for service '%s' failed: %s\033[m\n", name, err.Error())
			failed = true
			continue
		}
		_, _ = fmt.Fprintf(os.Stdout, "\033[32mSUCCESS: versioned migration files for '%s' (%s/%s) -> %s\033[m\n", name, dialect, tool.Name, dir)
	}
	if failed {
		return fmt.Errorf("one or more services failed to generate versioned migrations")
	}
	return nil
}

// versionedService 为单个服务生成版本化迁移文件,返回迁移目录。
func versionedService(cmd *cobra.Command, inspector *pkg.ModuleInspector, name, dialect string, tool migTool, dc struct{ Const, Driver string }) (string, error) {
	svcDir := filepath.Join(inspector.Root, "app", name, "service")
	entRoot := filepath.Join(svcDir, "internal", "data", "ent")
	migratePkg := filepath.Join(entRoot, "migrate", "migrate.go")

	if !hasEntSchema(inspector.Root, name) {
		return "", fmt.Errorf("no ent schema under %s; service is not ent-based", filepath.Join(entRoot, "schema"))
	}

	// 版本化 diff 依赖 feature 生成的 NamedDiff;产物缺失(或旧版无该函数)时
	// 先重新生成 ent 代码。
	data, err := os.ReadFile(migratePkg)
	if err != nil || !strings.Contains(string(data), "func NamedDiff(") {
		if err = ent.GenerateService(svcDir); err != nil {
			return "", fmt.Errorf("ent codegen required for versioned migrations: %w", err)
		}
	}

	migDir := filepath.Join(svcDir, "migrations")
	if err = os.MkdirAll(migDir, 0o755); err != nil {
		return "", err
	}

	importPath := strings.TrimSuffix(
		filepath.ToSlash(filepath.Join(inspector.ModPath, "app", name, "service", "internal", "data", "ent", "migrate")),
		"/",
	)
	program, err := renderVersionedProgram(versionedProgramArgs{
		ImportPath:   importPath,
		DialectConst: dc.Const,
		DriverImport: dc.Driver,
		DirCall:      tool.DirCall,
		FormatCall:   tool.FormatCall,
		UsesSQLTool:  tool.UsesSQLTool,
	})
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

	// 临时程序引用 atlas 的 migrate/sqltool 与 ent schema 包,这些 import 在
	// 流程开头的 tidy 之后才进入模块;此处再补一次 tidy 保证 go run 可解析。
	if err = pkg.GoModTidy(cmd.Context(), inspector.Root); err != nil {
		return "", err
	}

	// 与 dump 相同:go run 以服务目录为工作目录满足 internal 可见性;
	// 开发库 URL、迁移名与目录均经命令行参数传入,不内嵌进 Go 源码。
	absMigDir, err := filepath.Abs(migDir)
	if err != nil {
		return "", err
	}
	g := pkg.NewGoCmd(svcDir)
	runErr := g.Run("run", "./"+dumpToolDir, devURLFlag, sanitizeMigName(nameFlag), absMigDir)

	if runErr != nil {
		return "", fmt.Errorf("versioned diff program failed (kept at %s): %w", programPath, runErr)
	}
	_ = os.RemoveAll(tmpDir)
	return migDir, nil
}

// versionedProgramTmpl 版本化 diff 临时程序模板,遵循 ent 官方
// versioned migrations 文档的调用形态(WithDir + ModeReplay + WithDialect
// + WithFormatter)。运行期参数(URL/名称/目录)全部经 os.Args 传入。
var versionedProgramTmpl = template.Must(template.New("versioned").Parse(`// Code generated by gow migrate --versioned; DO NOT EDIT.

package main

import (
	"context"
	"fmt"
	"os"

	migrate "{{.ImportPath}}"
{{- if .UsesSQLTool}}
	"ariga.io/atlas/sql/sqltool"
{{- else}}
	atlas "ariga.io/atlas/sql/migrate"
{{- end}}
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"

	{{.DriverImport}}
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: versioned-diff <dev-db-url> <migration-name> <migrations-dir>")
		os.Exit(2)
	}
	ctx := context.Background()
	dir, err := {{.DirCall}}(os.Args[3])
	if err != nil {
		fmt.Fprintln(os.Stderr, "open migration dir:", err)
		os.Exit(1)
	}
	err = migrate.NamedDiff(ctx, os.Args[1], os.Args[2],
		schema.WithDir(dir),
		schema.WithMigrationMode(schema.ModeReplay),
		schema.WithDialect({{.DialectConst}}),
		schema.WithFormatter({{.FormatCall}}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "versioned diff:", err)
		os.Exit(1)
	}
}
`))

type versionedProgramArgs struct {
	ImportPath   string
	DialectConst string
	DriverImport string
	DirCall      string
	FormatCall   string
	UsesSQLTool  bool
}

func renderVersionedProgram(args versionedProgramArgs) (string, error) {
	var buf strings.Builder
	if err := versionedProgramTmpl.Execute(&buf, args); err != nil {
		return "", err
	}
	return buf.String(), nil
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
