package frontendgen

import "strings"

// reactRouteIconMap 服务模型名 -> Lucide 图标（精确匹配，react 版，与 element 版一致）
var reactRouteIconMap = map[string]string{
	"role":                    "lucide:shield-user",
	"permission":              "lucide:shield-ellipsis",
	"permissiongroup":         "lucide:shield-plus",
	"menu":                    "lucide:square-menu",
	"api":                     "lucide:route",
	"user":                    "lucide:user",
	"orgunit":                 "lucide:layers",
	"position":                "lucide:briefcase",
	"dicttype":                "lucide:library-big",
	"dictentry":               "lucide:list",
	"file":                    "lucide:file-search",
	"task":                    "lucide:list-todo",
	"loginpolicy":             "lucide:shield-x",
	"language":                "lucide:globe",
	"tenant":                  "lucide:building-2",
	"loginauditlog":           "lucide:scroll-text",
	"apiauditlog":             "lucide:file-text",
	"operationauditlog":       "lucide:file-clock",
	"dataaccessauditlog":      "lucide:database",
	"permissionauditlog":      "lucide:shield-alert",
	"internalmessage":         "lucide:message-square",
	"internalmessagecategory": "lucide:folder",
}

func reactInferIcon(modelName string) string {
	if icon, ok := reactRouteIconMap[strings.ToLower(modelName)]; ok {
		return icon
	}
	return "lucide:file"
}

// reactRouterCode 生成 router/modules/*.tsx（createLazyRoute + AppRouteObject 风格）
func reactRouterCode(services []*ParsedService, moduleConfig RouterModuleConfig) string {
	moduleKey := moduleConfig.ModuleKey
	moduleOrder := moduleConfig.ModuleOrder
	if moduleOrder == 0 {
		moduleOrder = 2000
	}

	var children []string
	order := 1
	for _, service := range services {
		crudPaths := GetCrudPaths(service)
		if crudPaths.List == nil {
			continue
		}

		serviceFileName := serviceToFileName(service.TagName)
		serviceKebab := strings.ReplaceAll(serviceFileName, "-", "")
		routeName := moduleKey + "-" + serviceKebab

		icon := reactInferIcon(service.ModelName)
		routeAuthority := ""
		if len(moduleConfig.Authority) > 0 {
			routeAuthority = "\n          // permission: '" + moduleConfig.Authority[0] + "',"
		}

		children = append(children, `      {
        name: '`+routeName+`',
        path: '`+serviceFileName+`',
        element: createLazyRoute(() => import('@/pages/app/`+moduleKey+`/`+serviceFileName+`')),
        meta: {
          title: 'routes:`+moduleKey+`-`+serviceFileName+`',
          icon: '`+icon+`',
          order: `+itoa(order)+`,`+routeAuthority+`
        },
      },`)

		order++
	}

	if len(children) == 0 {
		return "// " + moduleConfig.ModuleDisplayName + " - 没有可用的服务（需要 List 操作）"
	}

	moduleIcon := moduleConfig.ModuleIcon
	if moduleIcon == "" {
		moduleIcon = "lucide:folder"
	}
	parentAuthority := ""
	if len(moduleConfig.Authority) > 0 {
		parentAuthority = "\n      // permission: '" + moduleConfig.Authority[0] + "',"
	}

	var sb strings.Builder
	sb.WriteString(`import type { AppRouteObject } from '@/core/router/types';
import { createLazyRoute } from '@/core/router';

/**
 * ` + moduleConfig.ModuleDisplayName + `路由配置
 */
export const ` + moduleKey + `Routes: AppRouteObject[] = [
  {
    name: '` + moduleKey + `',
    path: '` + moduleKey + `',
    meta: {
      title: 'routes:` + moduleKey + `',
      icon: '` + moduleIcon + `',
      order: ` + itoa(moduleOrder) + `,
      keepAlive: true,` + parentAuthority + `
    },
    children: [
` + strings.Join(children, ",\n") + `
    ],
  },
];

export default ` + moduleKey + `Routes;
`)
	return sb.String()
}

// reactLocaleZhCN 生成 locales/zh-CN/_modules/*.json（按模块命名空间）
func reactLocaleZhCN(service *ParsedService) string {
	modelDesc := ExtractModuleName(service)

	entries := NewOrderedMap().
		Set("pageTitle", modelDesc+"管理").
		Set("moduleName", modelDesc).
		Set("serial", "序号")

	fieldEntries := BuildFieldEntriesZhCN(service.Fields, &FieldEntriesOptions{
		WithPlaceholder: true,
		WithRequired:    true,
	})
	for _, key := range fieldEntries.keys {
		entries.Set(key, fieldEntries.vals[key])
	}

	actionEntries := BuildActionMessagesZhCN(modelDesc)
	for _, key := range actionEntries.keys {
		entries.Set(key, actionEntries.vals[key])
	}

	if statusField := FindStatusField(service); statusField != nil && len(statusField.EnumValues) > 0 {
		entries.Set("statusMap", BuildStatusMapZhCN(statusField.EnumValues))
	}

	return entries.String() + "\n"
}

// reactLocaleEnUS 生成 locales/en-US/_modules/*.json
func reactLocaleEnUS(service *ParsedService) string {
	entries := NewOrderedMap().
		Set("pageTitle", service.ModelName+" Management").
		Set("moduleName", service.ModelName).
		Set("serial", "#")

	fieldEntries := BuildFieldEntriesEnUS(service.Fields, &FieldEntriesOptions{
		WithPlaceholder: true,
		WithRequired:    true,
	})
	for _, key := range fieldEntries.keys {
		entries.Set(key, fieldEntries.vals[key])
	}

	actionEntries := BuildActionMessagesEnUS(service.ModelName)
	for _, key := range actionEntries.keys {
		entries.Set(key, actionEntries.vals[key])
	}

	if statusField := FindStatusField(service); statusField != nil && len(statusField.EnumValues) > 0 {
		entries.Set("statusMap", BuildStatusMapEnUS(statusField.EnumValues))
	}

	return entries.String() + "\n"
}
