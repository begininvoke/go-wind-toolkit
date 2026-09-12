package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/ai"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI 助手（DDL 生成、微服务划分、代码审查）",
}

func addAIFlags(cmd *cobra.Command) {
	cmd.Flags().String("provider", "", "AI 服务商: openai | deepseek | anthropic | gemini | ollama | azure | custom（环境变量 GOWIND_AI_PROVIDER）")
	cmd.Flags().String("base-url", "", "API 基础地址（环境变量 GOWIND_AI_BASE_URL）")
	cmd.Flags().String("api-key", "", "API 密钥（环境变量 GOWIND_AI_API_KEY）")
	cmd.Flags().String("azure-api-version", "", "Azure OpenAI 部署的 api-version（环境变量 GOWIND_AI_AZURE_API_VERSION）")
	cmd.Flags().String("model", "", "模型名称（环境变量 GOWIND_AI_MODEL）")
	cmd.Flags().Float64("temperature", 0.7, "温度参数 (0.0-2.0)")
	cmd.Flags().Int("max-tokens", 0, "最大 token 数")
}

// buildAIService 构建 AI 服务:持久化配置为底(GUI 里配置一次即生效),
// 环境变量 GOWIND_AI_* 覆盖,命令行旗标最高优先级。不回写配置文件。
func buildAIService(cmd *cobra.Command) *ai.Service {
	cfg := ai.LoadConfig()
	if v := flagString(cmd, "provider", envOr("GOWIND_AI_PROVIDER", "")); v != "" {
		cfg.Provider = v
	}
	if v := flagString(cmd, "base-url", envOr("GOWIND_AI_BASE_URL", "")); v != "" {
		cfg.BaseURL = v
	}
	if v := flagString(cmd, "api-key", envOr("GOWIND_AI_API_KEY", "")); v != "" {
		cfg.APIKey = v
	}
	if v := flagString(cmd, "azure-api-version", envOr("GOWIND_AI_AZURE_API_VERSION", "")); v != "" {
		cfg.AzureAPIVersion = v
	}
	if v := flagString(cmd, "model", envOr("GOWIND_AI_MODEL", "")); v != "" {
		cfg.Model = v
	}
	if cmd.Flags().Changed("temperature") {
		if t, err := cmd.Flags().GetFloat64("temperature"); err == nil && t > 0 {
			cfg.Temperature = t
		}
	}
	if cmd.Flags().Changed("max-tokens") {
		if m, err := cmd.Flags().GetInt("max-tokens"); err == nil && m > 0 {
			cfg.MaxTokens = m
		}
	}

	svc := ai.NewService()
	svc.SetConfigTransient(cfg)
	return svc
}

var aiPresetsCmd = &cobra.Command{
	Use:   "presets",
	Short: "列出 AI 服务商预设",
	Run: func(cmd *cobra.Command, args []string) {
		emit(ai.GetProviderPresets())
	},
}

var aiTestCmd = &cobra.Command{
	Use:   "test",
	Short: "测试 AI 连通性",
	Run: func(cmd *cobra.Command, args []string) {
		result, err := buildAIService(cmd).TestConnection()
		if err != nil {
			fail(err)
		}
		emit(result)
	},
}

var aiDdlCmd = &cobra.Command{
	Use:   "ddl",
	Short: "根据需求文档生成 MySQL DDL",
	Run: func(cmd *cobra.Command, args []string) {
		reqFile := flagString(cmd, "requirements", "")
		if reqFile == "" {
			checkErr(fmt.Errorf("必须指定 --requirements（需求文档 Markdown/文本文件）"))
		}
		requirements, err := readFileContent(reqFile)
		if err != nil {
			checkErr(err)
		}

		svc := buildAIService(cmd)
		var result *ai.StepResult
		if boolFlag(cmd, "stream") {
			// 增量内容实时写 stderr,完整结果仍输出 stdout JSON。
			result, err = svc.GenerateDDLStream(requirements, func(delta string) {
				fmt.Fprint(os.Stderr, delta)
			})
		} else {
			result, err = svc.GenerateDDL(requirements)
		}
		if err != nil {
			fail(err)
		}
		emit(result)
	},
}

var aiPartitionCmd = &cobra.Command{
	Use:   "partition",
	Short: "根据 DDL 建议微服务划分",
	Run: func(cmd *cobra.Command, args []string) {
		ddlFile := flagString(cmd, "ddl", "")
		if ddlFile == "" {
			checkErr(fmt.Errorf("必须指定 --ddl（DDL 文件）"))
		}
		ddl, err := readFileContent(ddlFile)
		if err != nil {
			checkErr(err)
		}

		partitions, err := buildAIService(cmd).PartitionMicroservices(ddl)
		if err != nil {
			fail(err)
		}
		emit(partitions)
	},
}

var aiReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "AI 代码审查（Go/微服务/Kratos 维度）",
	Run: func(cmd *cobra.Command, args []string) {
		files := stringSliceFlag(cmd, "files")
		if len(files) == 0 {
			checkErr(fmt.Errorf("必须指定 --files（逗号分隔的文件路径）"))
		}

		contents := map[string]string{}
		for _, path := range files {
			content, err := readFileContent(path)
			if err != nil {
				checkErr(err)
			}
			contents[shortPath(path)] = content
		}

		svc := buildAIService(cmd)
		var result *ai.StepResult
		var err error
		if boolFlag(cmd, "stream") {
			result, err = svc.ReviewCodeStream(contents, func(delta string) {
				fmt.Fprint(os.Stderr, delta)
			})
		} else {
			result, err = svc.ReviewCode(contents)
		}
		if err != nil {
			fail(err)
		}
		emit(result)
	},
}

func shortPath(path string) string {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	if len(parts) > 3 {
		return strings.Join(parts[len(parts)-3:], "/")
	}
	return path
}

func init() {
	addAIFlags(aiTestCmd)
	addAIFlags(aiDdlCmd)
	addAIFlags(aiPartitionCmd)
	addAIFlags(aiReviewCmd)
	aiDdlCmd.Flags().String("requirements", "", "需求文档文件路径（必填）")
	aiDdlCmd.Flags().Bool("stream", false, "流式生成（增量内容实时输出到 stderr，完整结果仍输出 stdout JSON）")
	aiPartitionCmd.Flags().String("ddl", "", "DDL 文件路径（必填）")
	aiReviewCmd.Flags().StringSlice("files", nil, "要审查的文件路径（逗号分隔，必填）")
	aiReviewCmd.Flags().Bool("stream", false, "流式审查（增量内容实时输出到 stderr，完整结果仍输出 stdout JSON）")

	aiCmd.AddCommand(aiPresetsCmd, aiTestCmd, aiDdlCmd, aiPartitionCmd, aiReviewCmd)
}
