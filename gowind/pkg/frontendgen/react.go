package frontendgen

// generateReact react-antd 代码生成入口（对应 TS 版 react-antd/index.ts）
//
// 目标项目结构（src 下）：
//   api/hooks/*.ts                React Query hooks（经 @/api/client 的 apiClient 调 Service Client）
//   pages/app/{module}/...        ProTable 列表页 + DrawerForm 编辑抽屉 + constants
//   router/modules/*.tsx
//   locales/{lang}/_modules/*.json（按模块命名空间）
func generateReact(opts Options) []GeneratedFile {
	services := opts.resolveServices()
	serviceName := opts.serviceName()
	modulePathMap := opts.ModulePathMap
	generateTypes := opts.resolveTypes()

	var files []GeneratedFile

	hasType := func(t string) bool {
		for _, gt := range generateTypes {
			if gt == t {
				return true
			}
		}
		return false
	}

	for _, service := range services {
		fileName := serviceToFileName(service.TagName)
		modulePath := fileName
		if mp, ok := modulePathMap[fileName]; ok {
			modulePath = mp
		}
		modelPascal := toPascalCase(service.ModelName)

		if hasType(TypeHooks) {
			files = append(files, GeneratedFile{
				Path:        "api/hooks/" + fileName + ".ts",
				Content:     reactHooksCode(service, serviceName),
				Description: service.ModelName + " React Query Hooks",
				ServiceName: service.TagName,
				Type:        TypeHooks,
			})
		}

		if hasType(TypePage) {
			files = append(files, GeneratedFile{
				Path:        "pages/app/" + modulePath + "/index.tsx",
				Content:     reactPageCode(service, serviceName),
				Description: service.ModelName + " 列表页面",
				ServiceName: service.TagName,
				Type:        TypePage,
			})
		}

		if hasType(TypeDrawer) {
			files = append(files, GeneratedFile{
				Path:        "pages/app/" + modulePath + "/components/" + modelPascal + "Drawer.tsx",
				Content:     reactDrawerCode(service, serviceName),
				Description: service.ModelName + " 编辑抽屉",
				ServiceName: service.TagName,
				Type:        TypeDrawer,
			})

			if constantsCode := reactConstantsCode(service); constantsCode != "" {
				files = append(files, GeneratedFile{
					Path:        "pages/app/" + modulePath + "/constants.ts",
					Content:     constantsCode,
					Description: service.ModelName + " 常量定义",
					ServiceName: service.TagName,
					Type:        TypeDrawer,
				})
			}
		}

		if hasType(TypeLocale) {
			files = append(files, GeneratedFile{
				Path:        "locales/zh-CN/_modules/" + fileName + ".json",
				Content:     reactLocaleZhCN(service),
				Description: service.ModelName + " 中文国际化",
				ServiceName: service.TagName,
				Type:        TypeLocale,
			})
			files = append(files, GeneratedFile{
				Path:        "locales/en-US/_modules/" + fileName + ".json",
				Content:     reactLocaleEnUS(service),
				Description: service.ModelName + " 英文国际化",
				ServiceName: service.TagName,
				Type:        TypeLocale,
			})
		}
	}

	routerModules := opts.resolveRouterModules(services)
	if hasType(TypeRouter) && len(routerModules) > 0 {
		for _, moduleConfig := range routerModules {
			moduleServices := filterByTags(services, moduleConfig.ServiceTags)
			if len(moduleServices) == 0 {
				continue
			}

			files = append(files, GeneratedFile{
				Path:        "router/modules/" + moduleConfig.ModuleKey + ".tsx",
				Content:     reactRouterCode(moduleServices, moduleConfig),
				Description: moduleConfig.ModuleDisplayName + " 路由配置",
				ServiceName: moduleConfig.ModuleKey,
				Type:        TypeRouter,
			})
		}
	}

	return files
}
