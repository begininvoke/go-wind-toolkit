package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/devtools"
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "项目脚手架（从模板创建项目 / 添加服务）",
}

var scaffoldProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "从 go-wind-admin-template 模板创建新项目",
	Run: func(cmd *cobra.Command, args []string) {
		name := flagString(cmd, "name", "")
		module := flagString(cmd, "module", "")
		if name == "" || module == "" {
			checkErr(fmt.Errorf("必须指定 --name（项目名）与 --module（Go 模块路径）"))
		}

		opts := devtools.CreateProjectOptions{
			Name:      name,
			Module:    module,
			RepoURL:   flagString(cmd, "repo-url", ""),
			Branch:    flagString(cmd, "branch", ""),
			ParentDir: flagString(cmd, "parent-dir", "."),
		}

		result := devtools.CreateProject(context.Background(), opts)
		if !result.Success {
			fail(fmt.Errorf("%s\n%s", result.Error, result.Output))
		}
		emit(result)
	},
}

var scaffoldServiceCmd = &cobra.Command{
	Use:   "service",
	Short: "在现有项目中添加一个新服务",
	Run: func(cmd *cobra.Command, args []string) {
		name := flagString(cmd, "name", "")
		if name == "" {
			checkErr(fmt.Errorf("必须指定 --name（服务名）"))
		}

		servers := stringSliceFlag(cmd, "servers")
		if len(servers) == 0 {
			servers = []string{"grpc"}
		}
		dbClients := stringSliceFlag(cmd, "db-clients")
		if len(dbClients) == 0 {
			dbClients = []string{"ent"}
		}

		opts := devtools.AddServiceOptions{
			ServiceName: name,
			Servers:     servers,
			DbClients:   dbClients,
		}

		result := devtools.AddService(flagString(cmd, "path", "."), opts)
		if !result.Success {
			fail(fmt.Errorf("%s\n%s", result.Error, result.Output))
		}
		emit(result)
	},
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "开发工具（buf / ent / wire / go mod tidy）",
}

var devBufCmd = &cobra.Command{
	Use:   "buf",
	Short: "对 api 目录执行 buf generate",
	Run: func(cmd *cobra.Command, args []string) {
		result := devtools.RunBufGenerate(flagString(cmd, "path", "."))
		if !result.Success {
			fail(fmt.Errorf("%s\n%s", result.Error, result.Output))
		}
		emit(result)
	},
}

var devTidyCmd = &cobra.Command{
	Use:   "tidy",
	Short: "执行 go mod tidy",
	Run: func(cmd *cobra.Command, args []string) {
		result := devtools.RunGoModTidy(flagString(cmd, "path", "."))
		if !result.Success {
			fail(fmt.Errorf("%s\n%s", result.Error, result.Output))
		}
		emit(result)
	},
}

var devEntCmd = &cobra.Command{
	Use:   "ent [service]",
	Short: "执行 ent generate（无参数时对全部服务）",
	Run: func(cmd *cobra.Command, args []string) {
		path := flagString(cmd, "path", ".")
		var result *devtools.CommandResult
		if len(args) > 0 {
			result = devtools.RunEntGenerate(path, args[0])
		} else {
			result = devtools.RunEntGenerateAll(path)
		}
		if !result.Success {
			fail(fmt.Errorf("%s\n%s", result.Error, result.Output))
		}
		emit(result)
	},
}

var devWireCmd = &cobra.Command{
	Use:   "wire [service]",
	Short: "执行 wire 依赖注入生成（无参数时对全部服务）",
	Run: func(cmd *cobra.Command, args []string) {
		path := flagString(cmd, "path", ".")
		var result *devtools.CommandResult
		if len(args) > 0 {
			result = devtools.RunWire(path, args[0])
		} else {
			result = devtools.RunWireAll(path)
		}
		if !result.Success {
			fail(fmt.Errorf("%s\n%s", result.Error, result.Output))
		}
		emit(result)
	},
}

func init() {
	scaffoldProjectCmd.Flags().String("name", "", "项目名（必填）")
	scaffoldProjectCmd.Flags().String("module", "", "Go 模块路径（必填）")
	scaffoldProjectCmd.Flags().String("repo-url", "", "模板仓库地址（缺省 github，失败回退 gitee）")
	scaffoldProjectCmd.Flags().String("branch", "", "模板分支")
	scaffoldProjectCmd.Flags().String("parent-dir", ".", "父目录")
	scaffoldCmd.AddCommand(scaffoldProjectCmd)

	scaffoldServiceCmd.Flags().String("name", "", "服务名（必填）")
	scaffoldServiceCmd.Flags().StringSlice("servers", nil, "服务端类型（默认 [grpc]）")
	scaffoldServiceCmd.Flags().StringSlice("db-clients", nil, "数据库客户端（默认 [ent]）")
	scaffoldServiceCmd.Flags().String("path", ".", "项目根目录")
	scaffoldCmd.AddCommand(scaffoldServiceCmd)

	devBufCmd.Flags().String("path", ".", "项目根目录")
	devTidyCmd.Flags().String("path", ".", "项目根目录")
	devEntCmd.Flags().String("path", ".", "项目根目录")
	devWireCmd.Flags().String("path", ".", "项目根目录")
	devCmd.AddCommand(devBufCmd, devTidyCmd, devEntCmd, devWireCmd)
}
