package extract

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
	pkgExtract "github.com/tx7do/go-wind-toolkit/gowind/pkg/extract"
)

var (
	extractOrmType string
	extractKeepSrc bool
	extractObj     []string
	extractDryRun  bool
	extractYes     bool
)

// CmdExtract 提取命令
var CmdExtract = &cobra.Command{
	Use:   "extract <source-service> <target-service> --obj <model> [--obj <model>...]",
	Short: "extract service modules from one service to another",
	Long: `Extract service modules (schema, repo, service, wiring, server) from source service to target service. Registration follows the target service form (anchor injection or legacy wire sets).

This is used for microservice evolution — gradually splitting a monolithic service
into smaller, independently deployable services.

ORM type is auto-detected from source service directory structure.

Examples:
  gow extract admin user --obj role
  gow extract admin user --obj role --obj permission
  gow extract admin user --obj role,permission
  gow extract admin user --obj role --orm ent
  gow extract admin user --obj role --keep-source`,
	Args: cobra.ExactArgs(2),
	RunE: runExtract,
}

func init() {
	CmdExtract.Flags().StringArrayVarP(&extractObj, "obj", "o", nil, "object/model names to extract (comma-separated or repeated flag)")
	CmdExtract.Flags().StringVarP(&extractOrmType, "orm", "", "", "ORM type override: ent, gorm (auto-detected by default)")
	CmdExtract.Flags().BoolVarP(&extractKeepSrc, "keep-source", "", false, "Keep source files instead of deleting them")
	CmdExtract.Flags().BoolVarP(&extractDryRun, "dry-run", "n", false, "Preview all file actions (copy/modify/delete) without touching anything")
	CmdExtract.Flags().BoolVarP(&extractYes, "yes", "y", false, "Skip the deletion confirmation prompt (for scripts)")
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

func runExtract(cmd *cobra.Command, args []string) error {
	sourceService := strings.TrimSpace(args[0])
	targetService := strings.TrimSpace(args[1])

	if sourceService == "" || targetService == "" {
		return fmt.Errorf("source and target service names are required")
	}

	if sourceService == targetService {
		return fmt.Errorf("source and target service cannot be the same")
	}

	// 解析 --obj 参数（支持逗号分隔和重复 flag）
	var filtered []string
	for _, obj := range extractObj {
		for _, part := range strings.Split(obj, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				filtered = append(filtered, part)
			}
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("at least one object name is required, use --obj <name>")
	}

	inspector, err := pkg.NewModuleInspectorFromGo("")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	srcValid, err := pkg.IsValidServiceName(inspector.Root, sourceService)
	if err != nil {
		return fmt.Errorf("validate source service: %w", err)
	}
	if !srcValid {
		return fmt.Errorf("source service '%s' does not exist or is not valid", sourceService)
	}

	// 目标服务不存在时，extractor 会自动创建（ensureTargetService）
	_, _ = pkg.IsValidServiceName(inspector.Root, targetService)

	ormType := extractOrmType
	if ormType == "" {
		srcPath := fmt.Sprintf("%s/app/%s/service", inspector.Root, sourceService)
		ormType = pkgExtract.DetectOrmType(srcPath)
		if ormType == "" {
			return fmt.Errorf("cannot detect ORM type for service '%s', please specify with --orm", sourceService)
		}
		fmt.Printf("Auto-detected ORM type: %s\n", ormType)
	}

	projectName := extractProjectName(inspector.ModPath)

	opts := pkgExtract.Options{
		RootPath:      inspector.Root,
		ModulePath:    inspector.ModPath,
		ProjectName:   projectName,
		SourceService: sourceService,
		TargetService: targetService,
		Models:        filtered,
		OrmType:       ormType,
		KeepSource:    extractKeepSrc,
	}

	fmt.Printf("Extracting %d model(s) from [%s] to [%s] (ORM: %s)...\n",
		len(filtered), sourceService, targetService, ormType)

	extractor := pkgExtract.NewExtractor(opts)

	// 只读计划:预览 + 删除前确认的依据。
	plan, err := extractor.Plan()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}
	printPlan(plan)

	for _, warning := range plan.Warnings {
		_, _ = fmt.Fprintf(os.Stderr, "\033[33mWARNING: %s\033[m\n", warning)
	}

	if extractDryRun {
		fmt.Printf("\033[36m[DRY-RUN] preview only — no files were written or deleted.\033[m\n")
		return nil
	}

	// 删除是默认行为且不可逆:展示摘要并确认(--keep-source 非破坏性,无需确认)。
	if !extractKeepSrc && !extractYes && len(plan.DeletedFiles) > 0 {
		confirm := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("The %d source file(s) listed above will be DELETED from [%s]. Proceed?",
				len(plan.DeletedFiles), sourceService),
			Default: false,
		}
		if err = survey.AskOne(prompt, &confirm); err != nil {
			return fmt.Errorf("confirmation prompt unavailable (%v); re-run with --yes to skip it (or --dry-run to preview)", err)
		}
		if !confirm {
			fmt.Printf("Aborted — nothing was modified.\n")
			return nil
		}
	}

	if err = extractor.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	fmt.Printf("\033[32mExtraction completed successfully!\033[m\n")
	fmt.Printf("  Source: %s\n", sourceService)
	fmt.Printf("  Target: %s\n", targetService)
	fmt.Printf("  Models: %s\n", strings.Join(filtered, ", "))

	return nil
}

// printPlan 渲染提取计划摘要。
func printPlan(plan *pkgExtract.Plan) {
	if plan.TargetWillBeCreated {
		fmt.Printf("  \033[36m+ create target service scaffold: app/%s/service\033[m\n", plan.TargetService)
	}

	fmt.Printf("  Copy %d file(s):\n", len(plan.CopyFiles))
	for _, c := range plan.CopyFiles {
		flag := ""
		if c.Overwrite {
			flag = " \033[33m(overwrite)\033[m"
		}
		fmt.Printf("    %s\n      -> %s%s\n", c.Src, c.Dst, flag)
	}

	if len(plan.ModifiedFiles) > 0 {
		fmt.Printf("  Modify %d file(s):\n", len(plan.ModifiedFiles))
		for _, f := range plan.ModifiedFiles {
			fmt.Printf("    %s\n", f)
		}
	}

	if !plan.KeepSource && len(plan.DeletedFiles) > 0 {
		fmt.Printf("  \033[31mDelete %d source file(s) (use --keep-source to keep them):\033[m\n", len(plan.DeletedFiles))
		for _, f := range plan.DeletedFiles {
			fmt.Printf("    %s\n", f)
		}
	} else {
		fmt.Printf("  Source files kept (--keep-source).\n")
	}
}
