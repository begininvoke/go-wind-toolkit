package frontendgen

// generateVben vue-vben 代码生成入口（对应 TS 版 vue-vben/index.ts）
//
// 目标项目结构（apps/admin/src 下）：
//   api/composables/*.ts     Vue Query hooks（经 #/api/client 的 apiClient 调 Service Client）
//   views/app/{module}/...   VxeGrid 列表页 + useVbenDrawer/useVbenForm 编辑抽屉
//   router/routes/modules/app/*.ts
//   locales/langs/{lang}/page.json|menu.json（合并式片段）
func generateVben(opts Options) []GeneratedFile {
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
				Content:     vbenComposableCode(service, serviceName),
				Description: service.ModelName + " Vue Query Composable",
				ServiceName: service.TagName,
				Type:        TypeComposable,
			})
		}

		if hasType(TypePage) {
			files = append(files, GeneratedFile{
				Path:        "views/app/" + modulePath + "/index.vue",
				Content:     vbenPageCode(service, serviceName, modulePath),
				Description: service.ModelName + " 列表页面",
				ServiceName: service.TagName,
				Type:        TypePage,
			})
		}

		if hasType(TypeDrawer) {
			files = append(files, GeneratedFile{
				Path:        "views/app/" + modulePath + "/" + fileName + "-drawer.vue",
				Content:     vbenDrawerCode(service, serviceName),
				Description: service.ModelName + " 编辑抽屉",
				ServiceName: service.TagName,
				Type:        TypeDrawer,
			})
		}

		if hasType(TypeLocale) {
			files = append(files, GeneratedFile{
				Path:        "locales/langs/zh-CN/page." + fileName + ".json",
				Content:     vbenLocalePageZhCN(service),
				Description: service.ModelName + " 中文国际化（page.json 片段）",
				ServiceName: service.TagName,
				Type:        TypeLocale,
			})
			files = append(files, GeneratedFile{
				Path:        "locales/langs/en-US/page." + fileName + ".json",
				Content:     vbenLocalePageEnUS(service),
				Description: service.ModelName + " 英文国际化（page.json 片段）",
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
				Content:     vbenRouterCode(moduleServices, moduleConfig, modulePathMap),
				Description: moduleConfig.ModuleDisplayName + " 路由配置",
				ServiceName: moduleConfig.ModuleKey,
				Type:        TypeRouter,
			})

			if hasType(TypeLocale) {
				files = append(files, GeneratedFile{
					Path:        "locales/langs/zh-CN/menu." + moduleConfig.ModuleKey + ".json",
					Content:     vbenLocaleMenuZhCN(moduleConfig.ModuleKey, moduleConfig.ModuleDisplayName, moduleServices),
					Description: moduleConfig.ModuleDisplayName + " 菜单中文国际化（menu.json 片段）",
					ServiceName: moduleConfig.ModuleKey,
					Type:        TypeLocale,
				})
				files = append(files, GeneratedFile{
					Path:        "locales/langs/en-US/menu." + moduleConfig.ModuleKey + ".json",
					Content:     vbenLocaleMenuEnUS(moduleConfig.ModuleKey, moduleConfig.ModuleDisplayName, moduleServices),
					Description: moduleConfig.ModuleDisplayName + " 菜单英文国际化（menu.json 片段）",
					ServiceName: moduleConfig.ModuleKey,
					Type:        TypeLocale,
				})
			}
		}
	}

	return files
}

// filterByTags 按 tag 名过滤服务（保持原序）
func filterByTags(services []*ParsedService, tags []string) []*ParsedService {
	var filtered []*ParsedService
	for _, svc := range services {
		for _, tag := range tags {
			if svc.TagName == tag {
				filtered = append(filtered, svc)
				break
			}
		}
	}
	return filtered
}
