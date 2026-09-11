package generate

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
	sqlkratos "github.com/tx7do/go-wind-toolkit/gowind/pkg/sqlkratos"
)

// CmdGenerate represents the generate command
var CmdGenerate = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "generate CRUD code from database schema",
	Long:    "Generate complete Kratos microservice code (proto, ORM, service, server, wiring, config) from an existing database or SQL file. Module registration follows the target service form: anchor injection into hand-written wiring.go, or wire provider sets for legacy services. Example: gow generate",
	Run:     run,
}

var (
	genDSN           string
	genDriver        string
	genServiceName   string
	genOrmType       string
	genServers       []string
	genTables        []string
	genExcludeTables []string
	genModuleVersion string
	genProtoOnly     bool
	genSkipORM       bool
	genSkipConfig    bool
	genSkipMakefile  bool
	genSourceModule  string
	genDryRun        bool
)

func init() {
	CmdGenerate.Flags().StringVarP(&genDSN, "dsn", "", "", "Database source name (DSN), e.g. mysql://user:pass@tcp(localhost:3306)/dbname")
	CmdGenerate.Flags().StringVarP(&genDriver, "driver", "", "mysql", "Database driver: mysql, postgres")
	CmdGenerate.Flags().StringVarP(&genServiceName, "service", "", "", "Service name (module name)")
	CmdGenerate.Flags().StringVarP(&genOrmType, "orm", "", "ent", "ORM type: ent, gorm")
	CmdGenerate.Flags().StringArrayVarP(&genServers, "servers", "s", []string{"grpc"}, "Server types: grpc, rest")
	CmdGenerate.Flags().StringArrayVarP(&genTables, "tables", "t", nil, "Tables to include (default: all tables)")
	CmdGenerate.Flags().StringArrayVarP(&genExcludeTables, "exclude-tables", "", nil, "Tables to exclude")
	CmdGenerate.Flags().StringVarP(&genModuleVersion, "module-version", "", "v1", "API module version")
	CmdGenerate.Flags().BoolVarP(&genProtoOnly, "proto-only", "", false, "Only generate proto files")
	CmdGenerate.Flags().BoolVarP(&genSkipORM, "skip-orm", "", false, "Skip ORM code generation")
	CmdGenerate.Flags().BoolVarP(&genSkipConfig, "skip-config", "", false, "Skip config file generation")
	CmdGenerate.Flags().BoolVarP(&genSkipMakefile, "skip-makefile", "", false, "Skip Makefile generation")
	CmdGenerate.Flags().StringVarP(&genSourceModule, "source-module", "", "", "Source module name for REST service")
	CmdGenerate.Flags().BoolVarP(&genDryRun, "dry-run", "n", false, "Validate the data source, resolve tables and preview the plan without writing anything")
}

func run(cmd *cobra.Command, args []string) {
	// 交互式获取缺失参数
	if genDSN == "" {
		prompt := &survey.Input{
			Message: "Database DSN?",
			Help:    "Database connection string, e.g. mysql://user:pass@tcp(localhost:3306)/dbname",
		}
		if err := survey.AskOne(prompt, &genDSN); err != nil || genDSN == "" {
			return
		}
	}

	if genServiceName == "" {
		prompt := &survey.Input{
			Message: "Service name?",
			Help:    "The service/module name to generate code for.",
		}
		if err := survey.AskOne(prompt, &genServiceName); err != nil || genServiceName == "" {
			return
		}
	}

	// 获取项目信息
	inspector, err := pkg.NewModuleInspectorFromGo("")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return
	}

	projectName := extractProjectName(inspector.ModPath)
	outputPath := inspector.Root

	sourceModule := genSourceModule
	if sourceModule == "" {
		sourceModule = genServiceName
	}

	// 构建 DSN 前缀（如果用户没有提供 scheme）
	dsn := genDSN

	opts := sqlkratos.GeneratorOptions{
		Driver:           genDriver,
		Source:           dsn,
		IncludedTables:   genTables,
		ExcludedTables:   genExcludeTables,
		OutputPath:       outputPath,
		SourceModuleName: sourceModule,
		ModuleName:       genServiceName,
		ModuleVersion:    genModuleVersion,
		OrmType:          genOrmType,
		ProjectName:      projectName,
		ServiceName:      genServiceName,
		Servers:          genServers,
		UseRepo:          true,
		GenerateProto:    true,
		GenerateServer:   !genProtoOnly,
		GenerateService:  !genProtoOnly,
		GenerateORM:      !genProtoOnly && !genSkipORM,
		GenerateData:     !genProtoOnly,
		GenerateMain:     !genProtoOnly,
		GenerateConfig:   !genProtoOnly && !genSkipConfig,
		GenerateMakefile: !genProtoOnly && !genSkipMakefile,
	}

	fmt.Printf("Generating code for service [%s] with ORM [%s]...\n", genServiceName, genOrmType)

	if genDryRun {
		tables, terr := sqlkratos.PlanTables(cmd.Context(), opts)
		if terr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: resolve tables: %s\033[m\n", terr.Error())
			return
		}
		fmt.Printf("Plan for service [%s]:\n", genServiceName)
		fmt.Printf("  Source:      %s (driver: %s)\n", dsn, genDriver)
		fmt.Printf("  ORM:         %s\n", genOrmType)
		fmt.Printf("  Servers:     %s\n", strings.Join(genServers, ", "))
		fmt.Printf("  API version: %s\n", genModuleVersion)
		fmt.Printf("  Output:      %s\n", outputPath)
		var skips []string
		if genProtoOnly {
			skips = append(skips, "proto-only (no server/service/ORM/config/Makefile)")
		}
		if genSkipORM {
			skips = append(skips, "ORM")
		}
		if genSkipConfig {
			skips = append(skips, "config")
		}
		if genSkipMakefile {
			skips = append(skips, "Makefile")
		}
		if len(skips) > 0 {
			fmt.Printf("  Skipped:     %s\n", strings.Join(skips, ", "))
		}
		fmt.Printf("  Resolved %d table(s):\n", len(tables))
		for _, t := range tables {
			note := ""
			if len(t.Fields) == 0 {
				note = "  (no fields — will be skipped)"
			}
			fmt.Printf("    - %s (%d field(s))%s\n", t.Name, len(t.Fields), note)
		}
		fmt.Printf("\033[36m[DRY-RUN] preview only — nothing was written. Re-run without --dry-run to execute.\033[m\n")
		return
	}

	if err := sqlkratos.Generate(cmd.Context(), opts); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return
	}

	fmt.Printf("\033[32mService [%s] generated successfully!\033[m\n", genServiceName)
}

func extractProjectName(module string) string {
	if module == "" {
		return ""
	}
	if idx := len(module) - 1; idx >= 0 {
		for i := len(module) - 1; i >= 0; i-- {
			if module[i] == '/' {
				return module[i+1:]
			}
		}
	}
	return module
}
