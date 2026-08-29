package frontendgen

import "strings"

// zhToEnMap 中文描述到英文的翻译映射表（适用于所有前端框架的国际化代码生成）
var zhToEnMap = map[string]string{
	// 通用
	"ID": "ID",
	"名称": "Name",
	"编码": "Code",
	"描述": "Description",
	"状态": "Status",
	"排序": "Sort Order",
	"备注": "Remark",
	"类型": "Type",
	"标题": "Title",
	"内容": "Content",
	"创建时间": "Created At",
	"更新时间": "Updated At",
	"删除时间": "Deleted At",
	"创建者":  "Created By",
	"更新者":  "Updated By",

	// 用户相关
	"用户名": "Username",
	"昵称":   "Nickname",
	"真实姓名": "Real Name",
	"头像":   "Avatar",
	"邮箱":   "Email",
	"手机号":  "Mobile",
	"手机号码": "Mobile",
	"座机号":  "Telephone",
	"性别":   "Gender",
	"住址":   "Address",
	"个人描述": "Personal Description",
	"最后登录时间": "Last Login At",
	"最后登录IP": "Last Login IP",
	"锁定截止时间": "Locked Until",
	"密码":   "Password",

	// 组织相关
	"组织名称": "Organization Name",
	"组织ID": "Organization ID",
	"职位名称": "Position Name",
	"父级组织": "Parent Organization",
	"管理员":  "Admin",
	"负责人":  "Leader",

	// 权限相关
	"角色名称": "Role Name",
	"角色编码": "Role Code",
	"角色标识码": "Role Code",
	"角色类型": "Role Type",
	"受保护角色": "Is Protected",
	"权限点":  "Permission Point",
	"权限配置": "Permissions",
	"所属租户": "Tenant",
	"权限名称": "Permission Name",
	"权限编码": "Permission Code",
	"唯一编码": "Unique Code",

	// 字典相关
	"类型名称": "Type Name",
	"类型编码": "Type Code",
	"标签":   "Label",
	"值":    "Value",
	"数值":   "Numeric Value",
	"多语言配置": "I18n Configuration",
	"语言代码": "Language Code",
	"语言名称": "Language Name",

	// 语言
	"本地名称": "Native Name",
	"是否启用": "Is Enabled",
	"是否默认": "Is Default",

	// 文件相关
	"文件名": "File Name",
	"文件大小": "File Size",
	"文件类型": "File Type",
	"存储路径": "Storage Path",
	"MIME类型": "MIME Type",

	// 租户相关
	"租户名称": "Tenant Name",
	"租户ID": "Tenant ID",
	"租户编码": "Tenant Code",
	"联系人":  "Contact",
	"联系电话": "Contact Phone",
	"联系邮箱": "Contact Email",
	"域名":   "Domain",
	"有效期":  "Expired At",
	"成员数量": "Member Count",
	"订阅时间": "Subscription At",
	"订阅套餐": "Subscription Plan",
	"审核状态": "Audit Status",

	// 菜单相关
	"菜单名称": "Menu Name",
	"菜单路径": "Menu Path",
	"菜单图标": "Menu Icon",
	"组件路径": "Component Path",
	"重定向":  "Redirect",
	"是否外链": "Is External",
	"是否缓存": "Is KeepAlive",
	"是否可见": "Is Visible",
	"父级菜单": "Parent Menu",

	// 日志相关
	"请求方法": "Request Method",
	"请求路径": "Request Path",
	"请求参数": "Request Params",
	"响应状态码": "Response Status Code",
	"响应内容":  "Response Body",
	"IP地址":  "IP Address",
	"User-Agent": "User-Agent",
	"耗时":     "Duration",
	"是否成功":   "Is Success",
	"操作者":    "Operator",

	// 任务相关
	"任务类型": "Task Type",
	"任务数据": "Task Payload",
	"Cron表达式": "Cron Spec",
	"启用":      "Enable",
	"禁用":      "Inactive",

	// 策略相关
	"策略名称": "Policy Name",
	"最大登录尝试": "Max Login Attempts",
	"锁定时长(分钟)": "Lock Duration (min)",
	"密码最小长度":  "Min Password Length",
	"密码过期天数":  "Password Expire Days",

	// 消息相关
	"消息分类": "Category",
	"消息主题": "Subject",
	"发送者":  "Sender",
	"接收者":  "Receiver",
	"已读":   "Is Read",
	"发送时间": "Sent At",

	// 其他
	"是":    "Yes",
	"否":    "No",
	"序号":   "#",
	"图标":   "Icon",
	"路径":   "Path",
	"方法":   "Method",
	"分类名称": "Category Name",
	"分类编码": "Category Code",
	"消息内容": "Content",
	"消息类型": "Type",
	"消息状态": "Status",
	"排序值":  "Sort Order",
	"角色值":  "Role Code",
	"角色描述": "Role Description",
	"电子邮箱": "Email",
}

// translateToEn 将中文字段描述翻译为英文。
// 优先级: 1. 精确匹配 zhToEnMap; 2. 基于 fieldName 关键词推断; 3. PascalCase(fieldName)。
func translateToEn(zhText, fieldName string) string {
	if en, ok := zhToEnMap[zhText]; ok {
		return en
	}

	lower := strings.ToLower(fieldName)

	if strings.HasSuffix(lower, "name") {
		return toPascalCase(stripCaseInsensitiveSuffix(fieldName, "name")) + " Name"
	}
	if strings.HasSuffix(lower, "code") {
		return toPascalCase(stripCaseInsensitiveSuffix(fieldName, "code")) + " Code"
	}
	if strings.HasSuffix(lower, "id") {
		return toPascalCase(stripCaseInsensitiveSuffix(fieldName, "id")) + " ID"
	}
	if strings.HasSuffix(lower, "type") {
		return toPascalCase(stripCaseInsensitiveSuffix(fieldName, "type")) + " Type"
	}
	if strings.HasSuffix(lower, "at") || strings.HasSuffix(lower, "time") {
		return toPascalCase(fieldName) + " Time"
	}
	if strings.HasPrefix(lower, "is") {
		return toPascalCase(fieldName)
	}
	if strings.HasPrefix(lower, "has") {
		return toPascalCase(fieldName)
	}
	if strings.Contains(lower, "count") || strings.Contains(lower, "num") {
		return toPascalCase(fieldName) + " Count"
	}
	if strings.Contains(lower, "sort") {
		return "Sort Order"
	}
	if strings.Contains(lower, "status") {
		return "Status"
	}
	if strings.Contains(lower, "desc") {
		return "Description"
	}
	if strings.Contains(lower, "remark") {
		return "Remark"
	}
	if strings.Contains(lower, "url") {
		return toPascalCase(fieldName)
	}
	if strings.Contains(lower, "path") {
		return toPascalCase(fieldName)
	}

	return toPascalCase(fieldName)
}

// stripCaseInsensitiveSuffix 去掉大小写不敏感的尾部后缀
func stripCaseInsensitiveSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// FieldEntriesOptions 字段条目构建选项（nil 切片表示使用默认值）
type FieldEntriesOptions struct {
	// SkipFields 要跳过的字段名列表（默认 ["id"]）
	SkipFields []string
	// WithPlaceholder 生成 Placeholder 条目
	WithPlaceholder bool
	// WithRequired 生成 Required 校验提示条目
	WithRequired bool
	// SkipRequiredFields Required 跳过的字段名（默认 ["sortOrder","description","remark"]）
	SkipRequiredFields []string
}

func (o *FieldEntriesOptions) skipSet() map[string]bool {
	if o == nil || o.SkipFields == nil {
		return map[string]bool{"id": true}
	}
	set := map[string]bool{}
	for _, f := range o.SkipFields {
		set[f] = true
	}
	return set
}

func (o *FieldEntriesOptions) skipRequiredSet() map[string]bool {
	if o == nil || o.SkipRequiredFields == nil {
		return map[string]bool{"sortOrder": true, "description": true, "remark": true}
	}
	set := map[string]bool{}
	for _, f := range o.SkipRequiredFields {
		set[f] = true
	}
	return set
}

func (o *FieldEntriesOptions) withPlaceholder() bool {
	return o != nil && o.WithPlaceholder
}

func (o *FieldEntriesOptions) withRequired() bool {
	return o != nil && o.WithRequired
}

// BuildFieldEntriesZhCN 构建中文字段条目（保持字段文档序）
func BuildFieldEntriesZhCN(fields []ParsedField, options *FieldEntriesOptions) *OrderedMap {
	skipSet := options.skipSet()
	skipRequiredSet := options.skipRequiredSet()

	entries := NewOrderedMap()
	for i := range fields {
		field := &fields[i]
		if skipSet[field.Name] {
			continue
		}
		key := toSnakeCase(field.Name)
		desc := orDefault(field.Description, field.Name)
		entries.Set(key, desc)
		if options.withPlaceholder() {
			entries.Set(key+"_ph", "请输入"+desc)
		}
		if options.withRequired() && !field.IsBoolean && !skipRequiredSet[field.Name] {
			entries.Set(key+"_req", "请输入"+desc)
		}
	}
	return entries
}

// BuildFieldEntriesEnUS 构建英文字段条目（保持字段文档序）
func BuildFieldEntriesEnUS(fields []ParsedField, options *FieldEntriesOptions) *OrderedMap {
	skipSet := options.skipSet()
	skipRequiredSet := options.skipRequiredSet()

	entries := NewOrderedMap()
	for i := range fields {
		field := &fields[i]
		if skipSet[field.Name] {
			continue
		}
		key := toSnakeCase(field.Name)
		enName := translateToEn(field.Description, field.Name)
		entries.Set(key, enName)
		if options.withPlaceholder() {
			entries.Set(key+"_ph", "Enter "+strings.ToLower(enName))
		}
		if options.withRequired() && !field.IsBoolean && !skipRequiredSet[field.Name] {
			entries.Set(key+"_req", "Please enter "+strings.ToLower(enName))
		}
	}
	return entries
}

// BuildActionMessagesZhCN 生成中文操作消息（按钮、确认、成功/失败提示等）
func BuildActionMessagesZhCN(moduleDesc string) *OrderedMap {
	return NewOrderedMap().
		Set("action", "操作").
		Set("create", "新建"+moduleDesc).
		Set("edit", "编辑"+moduleDesc).
		Set("yes", "是").
		Set("no", "否").
		Set("deleteConfirmTitle", "确认删除").
		Set("deleteConfirmDesc", "确定要删除该{{moduleName}}吗？").
		Set("createSuccess", "创建成功").
		Set("updateSuccess", "更新成功").
		Set("deleteSuccess", "删除成功").
		Set("createFailed", "创建失败").
		Set("updateFailed", "更新失败").
		Set("deleteFailed", "删除失败").
		Set("fetchFailed", "获取数据失败")
}

// BuildActionMessagesEnUS 生成英文操作消息
func BuildActionMessagesEnUS(modelName string) *OrderedMap {
	return NewOrderedMap().
		Set("action", "Actions").
		Set("create", "New "+modelName).
		Set("edit", "Edit "+modelName).
		Set("yes", "Yes").
		Set("no", "No").
		Set("deleteConfirmTitle", "Confirm Delete").
		Set("deleteConfirmDesc", "Are you sure you want to delete this {{moduleName}}?").
		Set("createSuccess", "Created successfully").
		Set("updateSuccess", "Updated successfully").
		Set("deleteSuccess", "Deleted successfully").
		Set("createFailed", "Creation failed").
		Set("updateFailed", "Update failed").
		Set("deleteFailed", "Delete failed").
		Set("fetchFailed", "Failed to fetch data")
}

// BuildButtonMessagesZhCN 生成中文按钮消息（简化版，仅 create/update）
func BuildButtonMessagesZhCN(moduleDesc string) *OrderedMap {
	return NewOrderedMap().
		Set("create", "创建"+moduleDesc).
		Set("update", "更新"+moduleDesc)
}

// BuildButtonMessagesEnUS 生成英文按钮消息（简化版，仅 create/update）
func BuildButtonMessagesEnUS(moduleName string) *OrderedMap {
	return NewOrderedMap().
		Set("create", "Create "+moduleName).
		Set("update", "Update "+moduleName)
}

// ExtractModuleName 从服务描述中提取模块中文名，如 "角色管理服务" -> "角色"
func ExtractModuleName(service *ParsedService) string {
	name := service.Description
	name = strings.TrimSuffix(name, "管理")
	name = strings.TrimSuffix(name, "服务")
	name = strings.TrimSuffix(name, "查询")
	return strings.TrimSpace(name)
}

// FindStatusField 查找服务中的状态枚举字段
func FindStatusField(service *ParsedService) *ParsedField {
	for i := range service.Fields {
		f := &service.Fields[i]
		if f.IsEnum && strings.Contains(strings.ToLower(f.Name), "status") {
			return f
		}
	}
	return nil
}

// BuildStatusMapZhCN 生成状态枚举的中文映射，如 {ON: 启用, OFF: 禁用}
func BuildStatusMapZhCN(enumValues []string) *OrderedMap {
	m := NewOrderedMap()
	for _, v := range enumValues {
		if v == "ON" {
			m.Set(v, "启用")
		} else if v == "OFF" {
			m.Set(v, "禁用")
		} else {
			m.Set(v, v)
		}
	}
	return m
}

// BuildStatusMapEnUS 生成状态枚举的英文映射，如 {ON: Active, OFF: Inactive}
func BuildStatusMapEnUS(enumValues []string) *OrderedMap {
	m := NewOrderedMap()
	for _, v := range enumValues {
		if v == "ON" {
			m.Set(v, "Active")
		} else if v == "OFF" {
			m.Set(v, "Inactive")
		} else {
			m.Set(v, v)
		}
	}
	return m
}
