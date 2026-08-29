package frontendgen

import "strings"

// elementRouteIconMap 服务模型名 -> Lucide 图标（精确匹配，element 版）
var elementRouteIconMap = map[string]string{
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

func elementInferIcon(modelName string) string {
	if icon, ok := elementRouteIconMap[strings.ToLower(modelName)]; ok {
		return icon
	}
	return "lucide:file"
}

// elementRouterCode 生成 router/routes/modules/app/*.ts
func elementRouterCode(services []*ParsedService, moduleConfig RouterModuleConfig) string {
	moduleKey := moduleConfig.ModuleKey
	moduleOrder := moduleConfig.ModuleOrder
	if moduleOrder == 0 {
		moduleOrder = 2000
	}

	modulePascal := toPascalCase(moduleKey)
	moduleKebab := toKebabCase(moduleKey)

	var children []string
	order := 1
	for _, service := range services {
		crudPaths := GetCrudPaths(service)
		if crudPaths.List == nil {
			continue
		}

		serviceFileName := serviceToFileName(service.TagName)
		serviceKebab := toKebabCase(serviceFileName)
		servicePascal := toPascalCase(serviceFileName) + "Management"

		icon := elementInferIcon(service.ModelName)
		routeAuthority := ""
		if len(moduleConfig.Authority) > 0 {
			routeAuthority = "\n          authority: " + jsonString(moduleConfig.Authority) + ","
		}

		children = append(children, "\n      {\n"+
			"        path: \""+serviceKebab+"\",\n"+
			"        name: \""+servicePascal+"\",\n"+
			"        meta: {\n"+
			"          order: "+itoa(order)+",\n"+
			"          icon: \""+icon+"\",\n"+
			"          title: \"routes."+moduleKey+"."+serviceFileName+"\","+routeAuthority+"\n"+
			"        },\n"+
			"        component: () => import(\"@/pages/app/"+moduleKey+"/"+serviceFileName+"/index.vue\"),\n"+
			"      },")

		order++
	}

	moduleIcon := moduleConfig.ModuleIcon
	if moduleIcon == "" {
		moduleIcon = "lucide:folder"
	}
	parentAuthority := ""
	if len(moduleConfig.Authority) > 0 {
		parentAuthority = "\n      authority: " + jsonString(moduleConfig.Authority) + ","
	}

	firstRedirect := ""
	if len(services) > 0 {
		firstRedirect = toKebabCase(serviceToFileName(services[0].TagName))
	}

	var sb strings.Builder
	sb.WriteString(`import type { RouteRecordRaw } from "vue-router";
import { Layout } from "@/layouts";

const ` + moduleKey + `: RouteRecordRaw[] = [
  {
    path: "/` + moduleKebab + `",
    name: "` + modulePascal + `Management",
    component: Layout,
    redirect: "/` + moduleKebab + `/` + firstRedirect + `",
    meta: {
      order: ` + itoa(moduleOrder) + `,
      icon: "` + moduleIcon + `",
      title: "routes.` + moduleKey + `.moduleName",
      keepAlive: true,` + parentAuthority + `
    },
    children: [` + strings.Join(children, ",") + `
    ],
  },
];

export default ` + moduleKey + `;
`)
	return sb.String()
}

// elementLocaleZhCN 生成 locales/zh-CN/pages/*.json
func elementLocaleZhCN(service *ParsedService) string {
	moduleName := ExtractModuleName(service)

	entries := NewOrderedMap().
		Set("moduleName", moduleName)
	fieldEntries := BuildFieldEntriesZhCN(service.Fields, nil)
	for _, key := range fieldEntries.keys {
		entries.Set(key, fieldEntries.vals[key])
	}
	entries.Set("button", BuildButtonMessagesZhCN(moduleName))

	return entries.String() + "\n"
}

// elementLocaleEnUS 生成 locales/en-US/pages/*.json
func elementLocaleEnUS(service *ParsedService) string {
	entries := NewOrderedMap().
		Set("moduleName", service.ModelName)
	fieldEntries := BuildFieldEntriesEnUS(service.Fields, nil)
	for _, key := range fieldEntries.keys {
		entries.Set(key, fieldEntries.vals[key])
	}
	entries.Set("button", BuildButtonMessagesEnUS(service.ModelName))

	return entries.String() + "\n"
}
