// gowind-cli — go-wind-toolkit 的命令行入口
//
// 将 gowind-uiapp 桌面应用的全部能力（项目探测、数据库、后端/前端代码生成、
// AI 助手、远程配置、脚手架、开发工具）以非交互方式暴露，供人和 AI 调用。
//
// 约定:
//   - 结果输出到 stdout，日志输出到 stderr
//   - 默认输出缩进 JSON（人可读）；--json 输出单行紧凑 JSON（机器友好）
//   - 退出码: 0 成功 / 1 失败 / 2 用法错误
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

var jsonCompact bool

var rootCmd = &cobra.Command{
	Use:     "gowind-cli",
	Short:   "go-wind-toolkit 命令行工具（代码生成、数据库、AI、配置中心、脚手架、开发工具）",
	Version: version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func main() {
	rootCmd.PersistentFlags().BoolVar(&jsonCompact, "json", false, "以单行紧凑 JSON 输出结果（默认缩进 JSON）")

	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(backendCmd)
	rootCmd.AddCommand(frontendCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(devCmd)

	if err := rootCmd.Execute(); err != nil {
		fail(err)
	}
}

// emit 输出结果到 stdout（JSON）
func emit(v any) {
	var data []byte
	var err error
	if jsonCompact {
		data, err = json.Marshal(v)
	} else {
		data, err = json.MarshalIndent(v, "", "  ")
	}
	if err != nil {
		fail(err)
	}
	fmt.Println(string(data))
}

// fail 输出错误并以退出码 1 结束
func fail(err error) {
	if jsonCompact {
		fmt.Println(mustJSON(map[string]string{"error": err.Error()}))
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	}
	os.Exit(1)
}

func mustJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}

// logf 输出日志到 stderr
func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
