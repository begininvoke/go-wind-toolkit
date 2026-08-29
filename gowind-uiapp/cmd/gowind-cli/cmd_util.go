package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// envOr 取环境变量，空则用默认值
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// boolFlag 取布尔 flag 值
func boolFlag(cmd *cobra.Command, name string) bool {
	v, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false
	}
	return v
}

// stringSliceFlag 取字符串切片 flag 值
func stringSliceFlag(cmd *cobra.Command, name string) []string {
	v, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		return nil
	}
	return v
}

// checkErr 参数校验失败（用法错误，退出码 2）
func checkErr(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	fmt.Fprintf(os.Stderr, "运行 gowind-cli <命令> --help 查看用法\n")
	os.Exit(2)
}

// readFileContent 读取文件内容
func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件 %s 失败: %w", path, err)
	}
	return string(data), nil
}
