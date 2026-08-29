package frontendgen

import "strings"

// vbenLocalePageZhCN 生成中文 page.json 合并片段（缩进两格的 modelCamel 块）
func vbenLocalePageZhCN(service *ParsedService) string {
	modelCamel := toCamelCase(service.ModelName)
	moduleName := ExtractModuleName(service)

	fieldEntries := BuildFieldEntriesZhCN(service.Fields, &FieldEntriesOptions{
		SkipFields: []string{"id", "createdAt", "updatedAt"},
	})
	buttons := BuildButtonMessagesZhCN(moduleName)

	lines := []string{}
	lines = append(lines, `  "`+modelCamel+`": {`)
	lines = append(lines, `    "moduleName": "`+moduleName+`",`)
	for _, key := range fieldEntries.keys {
		lines = append(lines, `    "`+key+`": "`+entryText(fieldEntries, key)+`",`)
	}
	lines = append(lines, `    "button": {`)
	lines = append(lines, `      "create": "`+buttonText(buttons, "create")+`",`)
	lines = append(lines, `      "update": "`+buttonText(buttons, "update")+`"`)
	lines = append(lines, `    }`)
	lines = append(lines, `  }`)

	return strings.Join(lines, "\n")
}

// vbenLocalePageEnUS 生成英文 page.json 合并片段
func vbenLocalePageEnUS(service *ParsedService) string {
	modelCamel := toCamelCase(service.ModelName)
	moduleName := translateToEn(ExtractModuleName(service), modelCamel)

	fieldEntries := BuildFieldEntriesEnUS(service.Fields, &FieldEntriesOptions{
		SkipFields: []string{"id", "createdAt", "updatedAt"},
	})
	buttons := BuildButtonMessagesEnUS(moduleName)

	lines := []string{}
	lines = append(lines, `  "`+modelCamel+`": {`)
	lines = append(lines, `    "moduleName": "`+moduleName+`",`)
	for _, key := range fieldEntries.keys {
		lines = append(lines, `    "`+key+`": "`+entryText(fieldEntries, key)+`",`)
	}
	lines = append(lines, `    "button": {`)
	lines = append(lines, `      "create": "`+buttonText(buttons, "create")+`",`)
	lines = append(lines, `      "update": "`+buttonText(buttons, "update")+`"`)
	lines = append(lines, `    }`)
	lines = append(lines, `  }`)

	return strings.Join(lines, "\n")
}

// vbenLocaleMenuZhCN 生成中文 menu.json 合并片段（路由模块级别）
func vbenLocaleMenuZhCN(moduleKey, moduleDisplayName string, services []*ParsedService) string {
	lines := []string{}
	lines = append(lines, `  "`+moduleKey+`": {`)
	lines = append(lines, `    "moduleName": "`+moduleDisplayName+`",`)

	for _, service := range services {
		modelCamel := toCamelCase(service.ModelName)
		moduleName := ExtractModuleName(service)
		lines = append(lines, `    "`+modelCamel+`": "`+moduleName+`管理",`)
	}

	lines = append(lines, `  }`)
	return strings.Join(lines, "\n")
}

// vbenLocaleMenuEnUS 生成英文 menu.json 合并片段
func vbenLocaleMenuEnUS(moduleKey, moduleDisplayName string, services []*ParsedService) string {
	lines := []string{}
	lines = append(lines, `  "`+moduleKey+`": {`)
	lines = append(lines, `    "moduleName": "`+moduleDisplayName+`",`)

	for _, service := range services {
		modelCamel := toCamelCase(service.ModelName)
		enName := translateToEn(orDefault(service.Description, ""), modelCamel)
		lines = append(lines, `    "`+modelCamel+`": "`+enName+`",`)
	}

	lines = append(lines, `  }`)
	return strings.Join(lines, "\n")
}

func buttonText(buttons *OrderedMap, key string) string {
	if v, ok := buttons.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// entryText 取有序对象中指定键的字符串值
func entryText(entries *OrderedMap, key string) string {
	if v, ok := entries.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
