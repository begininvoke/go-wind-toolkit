package main

import (
	"fmt"

	"github.com/spf13/cobra"

	ce "github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/configexporter"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "远程配置中心导出（Consul / Nacos）",
}

var configTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "列出支持的配置中心类型",
	Run: func(cmd *cobra.Command, args []string) {
		emit(ce.GetSupportedTypes())
	},
}

var configServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "扫描项目中有配置文件的服务",
	Run: func(cmd *cobra.Command, args []string) {
		services, err := ce.GetServiceList(flagString(cmd, "path", "."))
		if err != nil {
			fail(err)
		}
		emit(services)
	},
}

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出服务配置到配置中心",
	Long: `将 app/<服务>/service/configs 下的配置文件导出到远程配置中心。
指定 --service 时只导出该服务，否则导出全部服务。Etcd 暂未实现。`,
	Run: func(cmd *cobra.Command, args []string) {
		typeName := flagString(cmd, "type", "")
		endpoint := flagString(cmd, "endpoint", "")
		projectName := flagString(cmd, "project", "")
		if typeName == "" || endpoint == "" || projectName == "" {
			checkErr(fmt.Errorf("必须指定 --type、--endpoint、--project"))
		}

		rc := ce.RemoteConfig{
			Type:        ce.ConfigType(typeName),
			Endpoint:    endpoint,
			ProjectName: projectName,
			Group:       flagString(cmd, "group", ""),
			Env:         flagString(cmd, "env", ""),
			NamespaceId: flagString(cmd, "namespace-id", ""),
		}
		if msg := rc.Validate(); msg != "" {
			checkErr(fmt.Errorf("%s", msg))
		}

		projectRoot := flagString(cmd, "path", ".")
		serviceName := flagString(cmd, "service", "")

		if boolFlag(cmd, "dry-run") {
			services, err := ce.GetServiceList(projectRoot)
			if err != nil {
				fail(err)
			}
			emit(map[string]any{
				"dryRun":   true,
				"type":     typeName,
				"endpoint": endpoint,
				"project":  projectName,
				"services": services,
			})
			return
		}

		if serviceName != "" {
			if err := ce.ExportOne(string(rc.Type), rc.Endpoint, rc.ProjectName, projectRoot, rc.Group, rc.Env, rc.NamespaceId, serviceName); err != nil {
				fail(err)
			}
			emit(map[string]any{"success": true, "service": serviceName})
			return
		}

		if err := ce.ExportAll(string(rc.Type), rc.Endpoint, rc.ProjectName, projectRoot, rc.Group, rc.Env, rc.NamespaceId); err != nil {
			fail(err)
		}
		emit(map[string]any{"success": true, "all": true})
	},
}

func init() {
	configServicesCmd.Flags().String("path", ".", "项目根目录")

	configExportCmd.Flags().String("type", "", "配置中心类型: consul | nacos（必填）")
	configExportCmd.Flags().String("endpoint", "", "配置中心地址，如 http://localhost:8500（必填）")
	configExportCmd.Flags().String("project", "", "项目名（配置 key 前缀，必填）")
	configExportCmd.Flags().String("group", "", "Nacos 分组")
	configExportCmd.Flags().String("env", "", "Nacos 环境")
	configExportCmd.Flags().String("namespace-id", "", "Nacos 命名空间")
	configExportCmd.Flags().String("service", "", "只导出指定服务（缺省全部）")
	configExportCmd.Flags().String("path", ".", "项目根目录")
	configExportCmd.Flags().Bool("dry-run", false, "只列出将导出的服务，不实际写入")

	configCmd.AddCommand(configTypesCmd, configServicesCmd, configExportCmd)
}
