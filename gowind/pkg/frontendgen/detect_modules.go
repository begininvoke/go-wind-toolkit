package frontendgen

import "strings"

// DetectRouterModules 按 basePath 第三段启发式检测路由模块分组
// （与 GUI 的 autoDetectRouterModules 逻辑一致）。
//
// 分组规则: basePath（如 /admin/v1/roles）去空段后取第 3 段（'roles'），
// 再取连字符前缀作为 groupKey，并归并到常见业务模块。
func DetectRouterModules(services []*ParsedService) []RouterModuleConfig {
	type groupState struct {
		config  RouterModuleConfig
		groupKey string
	}

	groupOrder := []string{}
	groups := map[string]*groupState{}

	for _, service := range services {
		parts := make([]string, 0, 4)
		for _, p := range strings.Split(service.BasePath, "/") {
			if p != "" {
				parts = append(parts, p)
			}
		}
		groupKey := "other"
		if len(parts) >= 3 {
			groupKey = strings.SplitN(parts[2], "-", 2)[0]
		}

		moduleKey := groupKey
		switch {
		case contains([]string{"audit", "login", "operation", "data"}, groupKey):
			moduleKey = "log"
		case contains([]string{"dict", "file", "language", "task", "loginP"}, groupKey):
			moduleKey = "system"
		case contains([]string{"permission", "role", "menu"}, groupKey):
			moduleKey = "permission"
		case contains([]string{"user", "org", "position"}, groupKey):
			moduleKey = "opm"
		case groupKey == "tenant":
			moduleKey = "tenant"
		case groupKey == "internal":
			moduleKey = "internalMessage"
		}

		if existing, ok := groups[moduleKey]; ok {
			existing.config.ServiceTags = append(existing.config.ServiceTags, service.TagName)
			continue
		}

		displayName := moduleDisplayName(service.Description)
		groups[moduleKey] = &groupState{
			groupKey: groupKey,
			config: RouterModuleConfig{
				ModuleKey:        moduleKey,
				ModuleDisplayName: displayName,
				ModuleIcon:       moduleIcon(groupKey),
				ModuleOrder:      2001 + len(groupOrder),
				ServiceTags:      []string{service.TagName},
			},
		}
		groupOrder = append(groupOrder, moduleKey)
	}

	modules := make([]RouterModuleConfig, 0, len(groupOrder))
	for _, key := range groupOrder {
		modules = append(modules, groups[key].config)
	}
	return modules
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// moduleDisplayName 从组内第一个服务描述推断模块中文名
func moduleDisplayName(desc string) string {
	if idx := strings.Index(desc, "管理"); idx >= 0 {
		desc = desc[:idx]
	}
	if idx := strings.Index(desc, "服务"); idx >= 0 {
		desc = desc[:idx]
	}
	if idx := strings.Index(desc, "查询"); idx >= 0 {
		desc = desc[:idx]
	}
	if idx := strings.Index(desc, "日志"); idx >= 0 {
		desc = desc[:idx] + "日志审计"
	}
	return strings.TrimSpace(desc)
}

var detectModuleIconMap = map[string]string{
	"api":       "lucide:route",
	"dict":      "lucide:library-big",
	"file":      "lucide:file-search",
	"login":     "lucide:shield-x",
	"permission": "lucide:shield-check",
	"opm":       "lucide:users",
	"user":      "lucide:user",
	"role":      "lucide:shield-user",
	"menu":      "lucide:square-menu",
	"tenant":    "lucide:building-2",
	"internal":  "lucide:message-square",
	"audit":     "lucide:scroll-text",
	"language":  "lucide:globe",
	"task":      "lucide:list-todo",
}

func moduleIcon(groupKey string) string {
	if icon, ok := detectModuleIconMap[groupKey]; ok {
		return icon
	}
	return "lucide:folder"
}
