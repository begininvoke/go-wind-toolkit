package frontendgen

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ==============================
// 解析后的业务模型
// ==============================

// CrudOperation API 操作类型
type CrudOperation string

const (
	OpList   CrudOperation = "list"
	OpGet    CrudOperation = "get"
	OpCreate CrudOperation = "create"
	OpUpdate CrudOperation = "update"
	OpDelete CrudOperation = "delete"
	OpOther  CrudOperation = "other"
)

var httpMethods = map[string]bool{"get": true, "post": true, "put": true, "delete": true, "patch": true}

// ParsedService 解析后的服务信息
type ParsedService struct {
	// TagName 服务标签名（如 RoleService）
	TagName string `json:"tagName"`
	// Description 服务描述
	Description string `json:"description"`
	// KebabName kebab-case 服务路径名（如 role）
	KebabName string `json:"kebabName"`
	// CamelName camelCase 服务名（如 role）
	CamelName string `json:"camelName"`
	// PascalName PascalCase 服务名（如 Role）
	PascalName string `json:"pascalName"`
	// ModelName 服务主模型名（如 Role）
	ModelName string `json:"modelName"`
	// ModelCamelName camelCase 模型名（如 role）
	ModelCamelName string `json:"modelCamelName"`
	// ClientGetterName ApiClient 上的 Service Client getter 名（如 roleService）
	ClientGetterName string `json:"clientGetterName"`
	// TypePrefix 生成代码中的类型前缀（如 permissionservicev1）
	TypePrefix string `json:"typePrefix"`
	// BasePath API 路径前缀（如 /admin/v1/roles）
	BasePath string `json:"basePath"`
	// Operations 该服务支持的操作（按文档序）
	Operations []ParsedOperation `json:"operations"`
	// Fields 主模型字段（按 schema 属性文档序）
	Fields []ParsedField `json:"fields"`
}

// ParsedOperation 解析后的操作信息
type ParsedOperation struct {
	Type        CrudOperation `json:"type"`
	Method      string        `json:"method"`
	Path        string        `json:"path"`
	Description string        `json:"description"`
	OperationID string        `json:"operationId"`
}

// ParsedField 解析后的字段信息
type ParsedField struct {
	Name        string   `json:"name"`
	TsType      string   `json:"tsType"`
	Description string   `json:"description"`
	IsEnum      bool     `json:"isEnum"`
	EnumValues  []string `json:"enumValues"`
	Format      string   `json:"format"`
	IsArray     bool     `json:"isArray"`
	IsBoolean   bool     `json:"isBoolean"`
	IsDate      bool     `json:"isDate"`
	IsInteger   bool     `json:"isInteger"`
}

// ==============================
// OpenAPI 规范（保序解析）
// ==============================

// Spec 解析后的 OpenAPI 规范。为与 TypeScript 版行为一致，
// paths、methods、properties 均按 YAML 文档序保存（Go map 无序）。
type Spec struct {
	Tags    []OpenApiTag
	Paths   []*PathEntry
	Schemas map[string]*Schema
}

// OpenApiTag OpenAPI tag 定义
type OpenApiTag struct {
	Name        string
	Description string
}

// PathEntry 单个 API 路径（方法按文档序）
type PathEntry struct {
	Path    string
	Methods []*MethodEntry
}

// MethodEntry 路径下的 HTTP 方法与操作
type MethodEntry struct {
	Method string // 小写 http 方法名
	Tags   []string
	// Description 操作描述
	Description string
	// OperationID operationId
	OperationID string
}

// Schema 组件 schema（属性按文档序）
type Schema struct {
	Properties []*Property
}

// Property schema 属性
type Property struct {
	Name        string
	Type        string
	Format      string
	Description string
	Enum        []string
	// ItemsType 数组元素类型（type: array 时）
	ItemsType string
}

// ParseOpenAPIYAML 解析 OpenAPI 3.0 YAML/JSON 文本
func ParseOpenAPIYAML(data []byte) (*Spec, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("解析 OpenAPI 失败: %w", err)
	}
	root := unwrapDoc(&doc)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("OpenAPI 文档必须是对象")
	}

	spec := &Spec{Schemas: map[string]*Schema{}}

	if tagsNode := mapValue(root, "tags"); tagsNode != nil && tagsNode.Kind == yaml.SequenceNode {
		for _, item := range tagsNode.Content {
			tag := OpenApiTag{
				Name:        scalarString(mapValue(item, "name")),
				Description: scalarString(mapValue(item, "description")),
			}
			spec.Tags = append(spec.Tags, tag)
		}
	}

	if pathsNode := mapValue(root, "paths"); pathsNode != nil && pathsNode.Kind == yaml.MappingNode {
		for i := 0; i+1 <= len(pathsNode.Content)-1; i += 2 {
			keyNode, valNode := pathsNode.Content[i], pathsNode.Content[i+1]
			if valNode.Kind != yaml.MappingNode {
				continue
			}
			entry := &PathEntry{Path: keyNode.Value}
			for j := 0; j+1 <= len(valNode.Content)-1; j += 2 {
				mKey, mVal := valNode.Content[j], valNode.Content[j+1]
				if !httpMethods[mKey.Value] || mVal.Kind != yaml.MappingNode {
					continue
				}
				me := &MethodEntry{
					Method:      mKey.Value,
					Description: scalarString(mapValue(mVal, "description")),
					OperationID: scalarString(mapValue(mVal, "operationId")),
				}
				if tagsNode := mapValue(mVal, "tags"); tagsNode != nil && tagsNode.Kind == yaml.SequenceNode {
					for _, t := range tagsNode.Content {
						me.Tags = append(me.Tags, t.Value)
					}
				}
				entry.Methods = append(entry.Methods, me)
			}
			spec.Paths = append(spec.Paths, entry)
		}
	}

	if compsNode := mapValue(root, "components"); compsNode != nil {
		if schemasNode := mapValue(compsNode, "schemas"); schemasNode != nil && schemasNode.Kind == yaml.MappingNode {
			for i := 0; i+1 <= len(schemasNode.Content)-1; i += 2 {
				keyNode, valNode := schemasNode.Content[i], schemasNode.Content[i+1]
				if valNode.Kind != yaml.MappingNode {
					continue
				}
				schema := &Schema{}
				if propsNode := mapValue(valNode, "properties"); propsNode != nil && propsNode.Kind == yaml.MappingNode {
					for j := 0; j+1 <= len(propsNode.Content)-1; j += 2 {
						pKey, pVal := propsNode.Content[j], propsNode.Content[j+1]
						if pVal.Kind != yaml.MappingNode {
							continue
						}
						prop := &Property{
							Name:        pKey.Value,
							Type:        scalarString(mapValue(pVal, "type")),
							Format:      scalarString(mapValue(pVal, "format")),
							Description: scalarString(mapValue(pVal, "description")),
						}
						if enumNode := mapValue(pVal, "enum"); enumNode != nil && enumNode.Kind == yaml.SequenceNode {
							for _, v := range enumNode.Content {
								prop.Enum = append(prop.Enum, v.Value)
							}
						}
						if itemsNode := mapValue(pVal, "items"); itemsNode != nil && itemsNode.Kind == yaml.MappingNode {
							prop.ItemsType = scalarString(mapValue(itemsNode, "type"))
						}
						schema.Properties = append(schema.Properties, prop)
					}
				}
				spec.Schemas[keyNode.Value] = schema
			}
		}
	}

	return spec, nil
}

// unwrapDoc 剥掉 yaml 文档包装节点
func unwrapDoc(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

// mapValue 取 mapping 节点指定 key 的值节点（不存在返回 nil）
func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 <= len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// scalarValue 取标量节点字符串（nil 安全）
func scalarString(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}

// ==============================
// 服务提取
// ==============================

// ExtractServices 从 OpenAPI 规范中提取所有服务（按首次出现的 tag 文档序）
func ExtractServices(spec *Spec) []*ParsedService {
	tagMap := map[string]OpenApiTag{}
	for _, tag := range spec.Tags {
		tagMap[tag.Name] = tag
	}

	type svcState struct {
		operations []ParsedOperation
	}
	tagOrder := []string{}
	tagStates := map[string]*svcState{}
	tagBasePaths := map[string]string{}

	for _, pathEntry := range spec.Paths {
		for _, me := range pathEntry.Methods {
			for _, tag := range me.Tags {
				state, ok := tagStates[tag]
				if !ok {
					state = &svcState{}
					tagStates[tag] = state
					tagOrder = append(tagOrder, tag)
				}
				state.operations = append(state.operations, ParsedOperation{
					Type:        detectOperationType(me.Method, pathEntry.Path, me.OperationID, me.Description),
					Method:      strings.ToUpper(me.Method),
					Path:        pathEntry.Path,
					Description: me.Description,
					OperationID: me.OperationID,
				})

				// 记录基础路径（最短的去参数路径）
				cleanPath := strings.TrimSuffix(pathEntry.Path, pathParamSuffix(pathEntry.Path))
				if existing, ok := tagBasePaths[tag]; !ok || len(cleanPath) < len(existing) {
					tagBasePaths[tag] = cleanPath
				}
			}
		}
	}

	services := make([]*ParsedService, 0, len(tagOrder))
	for _, tagName := range tagOrder {
		tag := tagMap[tagName]
		modelName := extractModelName(tagName)
		serviceBaseName := strings.TrimSuffix(tagName, "Service")

		services = append(services, &ParsedService{
			TagName:          tagName,
			Description:      tag.Description,
			KebabName:        toKebabCase(serviceBaseName),
			CamelName:        toCamelCase(serviceBaseName),
			PascalName:       serviceBaseName,
			ModelName:        modelName,
			ModelCamelName:   toCamelCase(modelName),
			ClientGetterName: toCamelCase(serviceBaseName) + "Service",
			TypePrefix:       extractTypePrefix(tagName),
			BasePath:         tagBasePaths[tagName],
			Operations:       tagStates[tagName].operations,
			Fields:           extractModelFields(spec, modelName),
		})
	}
	return services
}

// pathParamSuffix 返回路径末尾的 /{xxx} 部分（无参数返回空）
func pathParamSuffix(path string) string {
	idx := strings.LastIndex(path, "/{")
	if idx < 0 {
		return ""
	}
	if strings.HasSuffix(path, "}") {
		return path[idx:]
	}
	return ""
}

// detectOperationType 检测操作类型（operationId 优先，其次描述，最后 HTTP 方法）
func detectOperationType(method, path, operationID, description string) CrudOperation {
	opID := strings.ToLower(operationID)
	desc := strings.ToLower(description)

	if strings.Contains(opID, "_list") || strings.HasSuffix(opID, "list") {
		return OpList
	}
	if strings.Contains(opID, "_get") || strings.HasSuffix(opID, "get") {
		return OpGet
	}
	if strings.Contains(opID, "_create") || strings.HasSuffix(opID, "create") {
		return OpCreate
	}
	if strings.Contains(opID, "_update") || strings.HasSuffix(opID, "update") {
		return OpUpdate
	}
	if strings.Contains(opID, "_delete") || strings.HasSuffix(opID, "delete") {
		return OpDelete
	}

	// 注意: 与 TS 版一致，&& 优先于 ||
	if strings.Contains(desc, "列表") ||
		strings.Contains(desc, "查询") && method == "get" && !strings.Contains(path, "{") {
		return OpList
	}
	if strings.Contains(desc, "详情") || (method == "get" && strings.Contains(path, "{")) {
		return OpGet
	}
	if strings.Contains(desc, "创建") || method == "post" {
		return OpCreate
	}
	if strings.Contains(desc, "更新") || method == "put" {
		return OpUpdate
	}
	if strings.Contains(desc, "删除") || method == "delete" {
		return OpDelete
	}
	return OpOther
}

// extractModelName 从服务标签名提取模型名，如 RoleService -> Role
func extractModelName(tagName string) string {
	name := strings.TrimSuffix(tagName, "Service")

	specialCases := map[string]string{
		"DictType":                "DictType",
		"DictEntry":               "DictEntry",
		"InternalMessageCategory": "InternalMessageCategory",
		"InternalMessageRecipient": "InternalMessage",
		"InternalMessage":         "InternalMessage",
		"DataAccessAuditLog":      "DataAccessAuditLog",
		"ApiAuditLog":             "ApiAuditLog",
		"LoginAuditLog":           "LoginAuditLog",
		"OperationAuditLog":       "OperationAuditLog",
		"PermissionAuditLog":      "PermissionAuditLog",
		"PolicyEvaluation":        "PolicyEvaluationLog",
		"AdminPortal":             "AdminPortal",
		"Authentication":          "Auth",
		"FileTransfer":            "FileTransfer",
		"UserProfile":             "UserProfile",
		"PermissionGroup":         "PermissionGroup",
		"OrgUnit":                 "OrgUnit",
		"LoginPolicy":             "LoginPolicy",
	}

	if v, ok := specialCases[name]; ok {
		return v
	}
	return name
}

// extractModelFields 从 schema 定义中提取模型字段
func extractModelFields(spec *Spec, modelName string) []ParsedField {
	schema := spec.Schemas[modelName]
	if schema == nil {
		return nil
	}

	// schemaToFields: 跳过系统审计字段
	skipFields := map[string]bool{
		"createdBy": true, "updatedBy": true, "deletedBy": true,
		"createdAt": true, "updatedAt": true, "deletedAt": true,
	}

	fields := make([]ParsedField, 0, len(schema.Properties))
	for _, prop := range schema.Properties {
		if skipFields[prop.Name] {
			continue
		}
		f := resolveProperty(prop)
		fields = append(fields, ParsedField{
			Name:        prop.Name,
			TsType:      f.tsType,
			Description: orDefault(prop.Description, prop.Name),
			IsEnum:      f.isEnum,
			EnumValues:  f.enumValues,
			Format:      prop.Format,
			IsArray:     f.isArray,
			IsBoolean:   f.isBoolean,
			IsDate:      f.isDate,
			IsInteger:   f.isInteger,
		})
	}
	return fields
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

type resolvedProp struct {
	tsType     string
	isEnum     bool
	enumValues []string
	isArray    bool
	isBoolean  bool
	isDate     bool
	isInteger  bool
}

// resolveProperty 解析属性类型
func resolveProperty(prop *Property) resolvedProp {
	result := resolvedProp{tsType: "string"}

	if len(prop.Enum) > 0 {
		result.isEnum = true
		result.enumValues = prop.Enum
		quoted := make([]string, 0, len(prop.Enum))
		for _, v := range prop.Enum {
			quoted = append(quoted, "'"+v+"'")
		}
		result.tsType = strings.Join(quoted, " | ")
		return result
	}

	if prop.Type == "array" {
		result.isArray = true
		switch prop.ItemsType {
		case "integer":
			result.tsType = "number[]"
			result.isInteger = true
		case "string":
			result.tsType = "string[]"
		default:
			result.tsType = "any[]"
		}
		return result
	}

	switch prop.Type {
	case "integer":
		result.tsType = "number"
		result.isInteger = true
	case "boolean":
		result.tsType = "boolean"
		result.isBoolean = true
	case "number":
		result.tsType = "number"
	case "string":
		result.tsType = "string"
		if prop.Format == "date-time" {
			result.isDate = true
		}
	default:
		result.tsType = "any"
	}
	return result
}

// ==============================
// 服务类型前缀
// ==============================

// serviceTypePrefixes 服务 tag -> 领域包类型前缀映射。
//
// protoc-gen-typescript-http 生成的类型名以「消息定义所在的 proto 包」为前缀，
// 而聚合 OpenAPI 的 tag 只有服务名，因此按 go-wind-admin 的服务划分静态映射。
// 映射来源：api/generated/admin/service/v1 中各 Service Client 实际引用的类型。
var serviceTypePrefixes = map[string]string{
	"ApiService":                    "permissionservicev1",
	"MenuService":                   "permissionservicev1",
	"PermissionService":             "permissionservicev1",
	"PermissionGroupService":        "permissionservicev1",
	"RoleService":                   "permissionservicev1",
	"PolicyEvaluationLogService":    "permissionservicev1",
	"ApiAuditLogService":            "auditservicev1",
	"DataAccessAuditLogService":     "auditservicev1",
	"LoginAuditLogService":          "auditservicev1",
	"OperationAuditLogService":      "auditservicev1",
	"PermissionAuditLogService":     "auditservicev1",
	"AuthenticationService":         "authenticationservicev1",
	"LoginPolicyService":            "authenticationservicev1",
	"MfaService":                    "authenticationservicev1",
	"DictEntryService":              "dictservicev1",
	"DictTypeService":               "dictservicev1",
	"LanguageService":               "dictservicev1",
	"OrgUnitService":                "identityservicev1",
	"PlanService":                   "identityservicev1",
	"PlanModuleService":             "identityservicev1",
	"PlanQuotaService":              "identityservicev1",
	"PositionService":               "identityservicev1",
	"TenantService":                 "identityservicev1",
	"UserService":                   "identityservicev1",
	"UserProfileService":            "identityservicev1",
	"InternalMessageService":        "messageservicev1",
	"InternalMessageCategoryService": "messageservicev1",
	"InternalMessageRecipientService": "messageservicev1",
	"FileService":                   "storageservicev1",
	"FileTransferService":           "storageservicev1",
	"TaskService":                   "taskservicev1",
	"RedisCacheMonitorService":      "cacheservicev1",
}

// extractTypePrefix 提取服务类型前缀，如 RoleService -> permissionservicev1
func extractTypePrefix(tagName string) string {
	if prefix, ok := serviceTypePrefixes[tagName]; ok {
		return prefix
	}
	return strings.ReplaceAll(toKebabCase(strings.TrimSuffix(tagName, "Service")), "-", "")
}

// ==============================
// CRUD 工具
// ==============================

// HasFullCrud 判断服务是否有完整 CRUD 操作
func HasFullCrud(service *ParsedService) bool {
	types := map[CrudOperation]bool{}
	for _, op := range service.Operations {
		types[op.Type] = true
	}
	return types[OpList] && types[OpCreate] && types[OpUpdate] && types[OpDelete]
}

// CrudPaths 服务的主要 list/get/create/update/delete 操作（每类取首个）
type CrudPaths struct {
	List   *ParsedOperation
	Get    *ParsedOperation
	Create *ParsedOperation
	Update *ParsedOperation
	Delete *ParsedOperation
}

// GetCrudPaths 获取服务的主要 CRUD 路径
func GetCrudPaths(service *ParsedService) CrudPaths {
	var p CrudPaths
	for i := range service.Operations {
		op := &service.Operations[i]
		switch op.Type {
		case OpList:
			if p.List == nil {
				p.List = op
			}
		case OpGet:
			if p.Get == nil {
				p.Get = op
			}
		case OpCreate:
			if p.Create == nil {
				p.Create = op
			}
		case OpUpdate:
			if p.Update == nil {
				p.Update = op
			}
		case OpDelete:
			if p.Delete == nil {
				p.Delete = op
			}
		}
	}
	return p
}
