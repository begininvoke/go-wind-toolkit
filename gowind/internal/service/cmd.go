package service

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

// CmdService represents the service command
var CmdService = &cobra.Command{
	Use:     "service [name]",
	Aliases: []string{"svc"},
	Short:   "create a new service scaffold",
	Long:    "Create a new microservice inside the current workspace. Example: gow new service user",
	Args:    cobra.ExactArgs(1),
	Run:     run,
}

var (
	serviceName string
	Servers     []string
	DbClients   []string
	useWireDI   bool
	dryRun      bool
)

func init() {
	Servers = []string{"grpc"}
	DbClients = []string{"ent"}

	CmdService.Flags().StringArrayVarP(&Servers, "servers", "s", []string{"grpc"}, "Specify which server types to generate (grpc, rest, asynq, sse...)")
	CmdService.Flags().StringArrayVarP(&DbClients, "db-clients", "d", []string{"ent"}, "Specify which database clients to generate (gorm, ent, redis, clickhouse...)")
	CmdService.Flags().BoolVar(&useWireDI, "wire", false, "生成旧式 wire 依赖注入(wire.go + providers);默认生成手写装配 wiring.go")
	CmdService.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview the service layout without creating anything")
}

func extractProjectName(module string) string {
	module = strings.TrimSpace(module)
	if module == "" {
		return ""
	}

	if strings.Contains(module, "/") {
		parts := strings.Split(module, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			seg := strings.TrimSpace(parts[i])
			if seg != "" {
				return seg
			}
		}
	}

	return module
}

// splitList 展开每个元素内的逗号分隔写法("-s a,b" 等价 "-s a -s b")。
func splitList(list []string) []string {
	var out []string
	for _, item := range list {
		for _, part := range strings.Split(item, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func run(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		prompt := &survey.Input{
			Message: "What is service name?",
			Help:    "Created service name.",
		}
		err := survey.AskOne(prompt, &serviceName)
		if err != nil || serviceName == "" {
			return
		}
	} else {
		serviceName = args[0]
	}

	// 逗号分隔写法(-s grpc,rest)与重复 flag 写法(-s grpc -s rest)等价。
	Servers = splitList(Servers)
	DbClients = splitList(DbClients)

	inspector, err := pkg.NewModuleInspectorFromGo("")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return
	}

	servicePath := path.Join(inspector.Root, "/app/", serviceName, "/service")

	if pkg.IsDirExists(servicePath) {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: Service directory %s already exists\033[m\n", servicePath)
		return
	}

	if dryRun {
		wiringMode := "hand-written wiring.go (default)"
		if useWireDI {
			wiringMode = "legacy wire.go + providers"
		}
		fmt.Printf("Would create service [%s] at %s\n", serviceName, servicePath)
		fmt.Printf("  Servers:      %s\n", strings.Join(Servers, ", "))
		fmt.Printf("  DB clients:   %s\n", strings.Join(DbClients, ", "))
		fmt.Printf("  Wiring mode:  %s\n", wiringMode)
		fmt.Printf("  Layout:\n")
		fmt.Printf("    %s/\n", servicePath)
		fmt.Printf("      Makefile  configs/*.yaml\n")
		fmt.Printf("      cmd/server/main.go\n")
		if useWireDI {
			fmt.Printf("      cmd/server/wire.go  internal/{data,service}/providers/wire_set.go\n")
		} else {
			fmt.Printf("      cmd/server/wiring.go  (module registration anchors)\n")
		}
		for _, srv := range Servers {
			fmt.Printf("      internal/server/%s_server.go\n", srv)
		}
		fmt.Printf("      internal/service/  (business services)\n")
		for _, cli := range DbClients {
			fmt.Printf("      internal/data/client/%s_client.go\n", cli)
		}
		fmt.Printf("\033[36m[DRY-RUN] preview only — nothing was created.\033[m\n")
		return
	}

	_ = Generate(cmd.Context(), GeneratorOptions{
		GenerateMain:     true,
		GenerateServer:   true,
		GenerateService:  true,
		GenerateData:     true,
		GenerateMakefile: true,
		GenerateConfigs:  true,

		ProjectName:   extractProjectName(inspector.ModPath),
		ProjectModule: inspector.ModPath,
		ServiceName:   serviceName,

		Servers:   Servers,
		DbClients: DbClients,
		UseWireDI: useWireDI,

		OutputPath: inspector.Root,
	})
}
