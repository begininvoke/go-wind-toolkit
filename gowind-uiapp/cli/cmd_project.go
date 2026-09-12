package cli

import (
	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/detect"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "项目探测",
}

var projectInspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "探测项目信息（模块路径、服务列表、API 目录）",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := flagString(cmd, "path", ".")
		info, err := detect.NewProjectDetector().Detect(path)
		if err != nil {
			return err
		}
		emit(info)
		return nil
	},
}

func init() {
	projectInspectCmd.Flags().String("path", ".", "项目根目录（默认当前目录）")
	projectCmd.AddCommand(projectInspectCmd)
}

// flagString 取字符串 flag 值
func flagString(cmd *cobra.Command, name, def string) string {
	v, err := cmd.Flags().GetString(name)
	if err != nil || v == "" {
		return def
	}
	return v
}
