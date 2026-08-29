package frontendgen

// generateElement vue-element 代码生成入口（对应 TS 版 vue-element/index.ts）
//
// 目标项目结构（src 下）：
//   api/composables/*.ts        Vue Query hooks（经 @/api/client 的 apiClient 调 Service Client）
//   pages/app/{module}/...      ProPage 列表页 + ElDrawer 编辑抽屉
//   router/routes/modules/app/*.ts
//   locales/{lang}/pages/*.json
func generateElement(opts Options) []GeneratedFile {
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

		if hasType(TypeComposable) {
			files = append(files, GeneratedFile{
				Path:        "api/composables/" + fileName + ".ts",
				Content:     elementComposableCode(service, serviceName),
				Description: service.ModelName + " Vue Query Composable",
				ServiceName: service.TagName,
				Type:        TypeComposable,
			})
		}

		if hasType(TypePage) {
			files = append(files, GeneratedFile{
				Path:        "pages/app/" + modulePath + "/index.vue",
				Content:     elementPageCode(service, serviceName, modulePath),
				Description: service.ModelName + " 列表页面",
				ServiceName: service.TagName,
				Type:        TypePage,
			})
		}

		if hasType(TypeDrawer) {
			files = append(files, GeneratedFile{
				Path:        "pages/app/" + modulePath + "/" + fileName + "-drawer.vue",
				Content:     elementDrawerCode(service),
				Description: service.ModelName + " 编辑抽屉",
				ServiceName: service.TagName,
				Type:        TypeDrawer,
			})
		}

		if hasType(TypeLocale) {
			files = append(files, GeneratedFile{
				Path:        "locales/zh-CN/pages/" + fileName + ".json",
				Content:     elementLocaleZhCN(service),
				Description: service.ModelName + " 中文国际化",
				ServiceName: service.TagName,
				Type:        TypeLocale,
			})
			files = append(files, GeneratedFile{
				Path:        "locales/en-US/pages/" + fileName + ".json",
				Content:     elementLocaleEnUS(service),
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
				Path:        "router/routes/modules/app/" + moduleConfig.ModuleKey + ".ts",
				Content:     elementRouterCode(moduleServices, moduleConfig),
				Description: moduleConfig.ModuleDisplayName + " 路由配置",
				ServiceName: moduleConfig.ModuleKey,
				Type:        TypeRouter,
			})
		}
	}

	return files
}
