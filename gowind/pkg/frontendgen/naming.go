// Package frontendgen 从 OpenAPI 3.0 规范生成前端 CRUD 代码
// (composable/hooks、列表页、编辑抽屉、路由、国际化)，
// 支持 vue-element / vue-vben / react-antd 三种目标框架。
//
// 本包由 gowind-uiapp 的 TypeScript 生成器移植而来，
// 以 testdata/golden 黄金样本保证行为一致。
package frontendgen

import (
	"regexp"
	"strings"
)

var upperCaseRe = regexp.MustCompile(`([A-Z])`)

// toCamelCase 首字母小写: "Role" -> "role", "RoleName" -> "roleName"
func toCamelCase(str string) string {
	if str == "" {
		return ""
	}
	return strings.ToLower(str[:1]) + str[1:]
}

// toPascalCase 首字母大写: "role" -> "Role", "roleName" -> "RoleName"
func toPascalCase(str string) string {
	if str == "" {
		return ""
	}
	return strings.ToUpper(str[:1]) + str[1:]
}

// toSnakeCase camelCase/PascalCase -> snake_case: "roleName" -> "role_name"
func toSnakeCase(str string) string {
	return upperCaseRe.ReplaceAllStringFunc(str, func(m string) string {
		return "_" + strings.ToLower(m)
	})
}

// toKebabCase camelCase/PascalCase -> kebab-case: "DictType" -> "dict-type"
func toKebabCase(str string) string {
	kebab := strings.ToLower(upperCaseRe.ReplaceAllString(str, "-${1}"))
	return strings.TrimPrefix(kebab, "-")
}

// serviceToFileName 服务 tag 名转文件名: "RoleService" -> "role", "DictTypeService" -> "dict-type"
func serviceToFileName(tagName string) string {
	return toKebabCase(strings.TrimSuffix(tagName, "Service"))
}
