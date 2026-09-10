package build

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

var (
	outputDir  string
	targetOS   []string
	targetArch []string
)

// CmdBuild build 命令:编译单个、多个或全部服务,支持 GOOS/GOARCH 交叉编译。
var CmdBuild = &cobra.Command{
	Use:   "build [service...]",
	Short: "Build one, several, or all project services into binaries (cross-compilation supported)",
	Long: `Build services of the project into standalone binaries.

With no arguments every service under app/ is built; naming one or more services
builds just those. Each service's cmd/server main package is compiled.

The --os and --arch flags accept comma separated lists; every GOOS×GOARCH
combination is built (cross compilation), e.g. --os linux,windows --arch amd64.
Cross-compiled binaries carry a _<goos>_<goarch> filename suffix so the
combinations never overwrite each other.

By default binaries are written into each service's bin/ directory; --out
redirects all output into a single directory instead.`,
	Run: Run,
}

func init() {
	CmdBuild.Flags().StringVarP(&outputDir, "out", "o", "", "output directory for all built binaries (default: each service's bin/)")
	CmdBuild.Flags().StringArrayVar(&targetOS, "os", nil, "comma separated target GOOS list for cross compilation")
	CmdBuild.Flags().StringArrayVar(&targetArch, "arch", nil, "comma separated target GOARCH list for cross compilation")
}

// buildTarget 一次构建目标。
type buildTarget struct {
	goos     string
	goarch   string
	explicit bool
}

// Run 为 cobra 的 Run 回调:编译目标服务/目标平台组合。
func Run(cmd *cobra.Command, args []string) {
	inspector, err := pkg.NewModuleInspectorFromGo("")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return
	}

	// 先在模块根目录运行 `go mod tidy`
	if err = pkg.GoModTidy(cmd.Context(), inspector.Root); err != nil {
		return
	}

	names, err := resolveServiceNames(inspector.Root, args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return
	}

	targets, err := resolveTargets()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return
	}

	failed := false
	for _, name := range names {
		serviceDir := filepath.Join(inspector.Root, "app", name, "service")
		for _, target := range targets {
			outPath, err := ResolveOutputPath(name, serviceDir, target.goos, target.goarch, target.explicit)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
				failed = true
				continue
			}
			if err = BuildBinary(serviceDir, outPath, target.goos, target.goarch); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: build for service '%s' (%s/%s) failed: %s\033[m\n", name, target.goos, target.goarch, err.Error())
				failed = true
				continue
			}
			_, _ = fmt.Fprintf(os.Stdout, "\033[32mSUCCESS: built '%s' (%s/%s) -> %s\033[m\n", name, target.goos, target.goarch, outPath)
		}
	}
	if failed {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: one or more build targets failed\033[m\n")
	}
}

// resolveServiceNames 解析目标服务列表:无参数时枚举全部有效服务,否则逐个校验。
func resolveServiceNames(root string, args []string) ([]string, error) {
	if len(args) == 0 {
		names, err := pkg.ListServiceNames(root)
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("no valid services found under %s", filepath.Join(root, "app"))
		}
		return names, nil
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

// resolveTargets 解析构建目标组合。未指定 os/arch 时为单一原生目标;
// 指定任一即进入显式模式,做 GOOS×GOARCH 全组合。
func resolveTargets() ([]buildTarget, error) {
	oss := splitFlagList(targetOS)
	archs := splitFlagList(targetArch)

	if len(oss) == 0 && len(archs) == 0 {
		return []buildTarget{{goos: runtime.GOOS, goarch: runtime.GOARCH, explicit: false}}, nil
	}
	if len(oss) == 0 {
		oss = []string{runtime.GOOS}
	}
	if len(archs) == 0 {
		archs = []string{runtime.GOARCH}
	}

	targets := make([]buildTarget, 0, len(oss)*len(archs))
	for _, goos := range oss {
		for _, goarch := range archs {
			targets = append(targets, buildTarget{goos: goos, goarch: goarch, explicit: true})
		}
	}
	return targets, nil
}

// splitFlagList 展开逗号分隔的旗标取值列表。
func splitFlagList(list []string) []string {
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

// ResolveOutputPath 决定产物路径:默认各服务 bin/,--out 时集中到单目录;
// 交叉目标带 _<goos>_<goarch> 后缀,GOOS=windows 追加 .exe。
// 供 build 命令与 run 命令的多服务模式共用同一命名规则。
func ResolveOutputPath(name string, serviceDir string, goos string, goarch string, explicit bool) (string, error) {
	outDir := filepath.Join(serviceDir, "bin")
	if outputDir != "" {
		absDir, err := filepath.Abs(outputDir)
		if err != nil {
			return "", err
		}
		outDir = absDir
	} else {
		absDir, err := filepath.Abs(outDir)
		if err != nil {
			return "", err
		}
		outDir = absDir
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	binary := name
	if explicit {
		binary = fmt.Sprintf("%s_%s_%s", name, goos, goarch)
	}
	if goos == "windows" {
		binary += ".exe"
	}
	return filepath.Join(outDir, binary), nil
}

// BuildBinary 编译服务主程序为独立二进制。goos/goarch 为目标平台,
// 与宿主一致即原生构建。输出路径须为绝对路径。
func BuildBinary(serviceDir string, outPath string, goos string, goarch string) error {
	g := pkg.NewGoCmd(serviceDir)
	g.Env = []string{
		"GOOS=" + goos,
		"GOARCH=" + goarch,
	}
	return g.Run("build", "-o", outPath, "./cmd/server")
}
