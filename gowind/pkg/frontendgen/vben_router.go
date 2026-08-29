package frontendgen

import (
	"encoding/json"
	"strconv"
	"strings"
)

// vbenRouteIconMap 根据模型名推断 Lucide 图标（vben 版，按序包含匹配）
var vbenRouteIconMap = []struct{ key, icon string }{
	{"user", "lucide:user"},
	{"role", "lucide:shield-user"},
	{"permission", "lucide:shield-ellipsis"},
	{"menu", "lucide:square-menu"},
	{"api", "lucide:route"},
	{"org", "lucide:building"},
	{"position", "lucide:briefcase"},
	{"tenant", "lucide:building-2"},
	{"dict", "lucide:library-big"},
	{"file", "lucide:file-search"},
	{"task", "lucide:list-todo"},
	{"language", "lucide:globe"},
	{"log", "lucide:scroll-text"},
	{"message", "lucide:message-square"},
	{"policy", "lucide:shield-x"},
}

func vbenInferIcon(modelName string) string {
	lower := strings.ToLower(modelName)
	for _, entry := range vbenRouteIconMap {
		if strings.Contains(lower, entry.key) {
			return entry.icon
		}
	}
	return "lucide:folder"
}

// vbenRouterCode 生成 router/routes/modules/app/*.ts
func vbenRouterCode(services []*ParsedService, moduleConfig RouterModuleConfig, modulePathMap map[string]string) string {
	moduleKey := moduleConfig.ModuleKey
	moduleOrder := moduleConfig.ModuleOrder
	if moduleOrder == 0 {
		moduleOrder = 2001
	}

	// 子路由
	var children []string
	for index, service := range services {
		modelPascal := toPascalCase(service.ModelName)
		modelCamel := toCamelCase(service.ModelName)
		fileName := serviceToFileName(service.TagName)
		modulePath := fileName
		if mp, ok := modulePathMap[fileName]; ok {
			modulePath = mp
		}
		routeName := modelPascal + "Management"
		path := fileName + "s"

		icon := vbenInferIcon(service.ModelName)
		auth := ""
		if len(moduleConfig.Authority) > 0 {
			auth = "\n          authority: " + jsonString(moduleConfig.Authority) + ","
		}

		children = append(children, "      {\n"+
			"        path: '"+path+"',\n"+
			"        name: '"+routeName+"',\n"+
			"        meta: {\n"+
			"          order: "+itoa(index+1)+",\n"+
			"          icon: '"+icon+"',\n"+
			"          title: $t('menu."+moduleKey+"."+modelCamel+"'),"+auth+"\n"+
			"        },\n"+
			"        component: () => import('#/views/app/"+modulePath+"/index.vue'),\n"+
			"      },")
	}

	parentIcon := moduleConfig.ModuleIcon
	if parentIcon == "" {
		parentIcon = "lucide:folder"
	}
	parentAuth := ""
	if len(moduleConfig.Authority) > 0 {
		parentAuth = "\n      authority: " + jsonString(moduleConfig.Authority) + ","
	}
	parentPath := "/" + toKebabCase(moduleKey)
	firstChildPath := "index"
	if len(services) > 0 {
		firstChildPath = serviceToFileName(services[0].TagName) + "s"
	}

	var sb strings.Builder
	sb.WriteString(`import type { RouteRecordRaw } from 'vue-router';

import { BasicLayout } from '#/layouts';
import { $t } from '#/locales';

const ` + moduleKey + `: RouteRecordRaw[] = [
  {
    path: '` + parentPath + `',
    name: '` + toPascalCase(moduleKey) + `Management',
    component: BasicLayout,
    redirect: '` + parentPath + `/` + firstChildPath + `',
    meta: {
      order: ` + itoa(moduleOrder) + `,
      icon: '` + parentIcon + `',
      title: $t('menu.` + moduleKey + `.moduleName'),
      keepAlive: true,` + parentAuth + `
    },
    children: [
` + strings.Join(children, "\n\n") + `
    ],
  },
];

export default ` + moduleKey + `;
`)
	return sb.String()
}

// jsonString 序列化为与 JS JSON.stringify 一致的单行 JSON
func jsonString(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
