package frontendgen

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// OrderedMap 保持插入序的 JSON 对象，渲染结果与 JS JSON.stringify(obj, null, 2)
// 逐字节一致（不转义非 ASCII 与 HTML 字符），供国际化文件生成使用。
type OrderedMap struct {
	keys []string
	vals map[string]any // string 或 *OrderedMap
}

// NewOrderedMap 创建有序对象
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{vals: map[string]any{}}
}

// Set 追加键值（值: string 或 *OrderedMap）
func (m *OrderedMap) Set(key string, val any) *OrderedMap {
	if _, exists := m.vals[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.vals[key] = val
	return m
}

// Get 取值
func (m *OrderedMap) Get(key string) (any, bool) {
	v, ok := m.vals[key]
	return v, ok
}

// Len 键数量
func (m *OrderedMap) Len() int {
	if m == nil {
		return 0
	}
	return len(m.keys)
}

// MarshalJSON 实现 json.Marshaler，按键插入序渲染
func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	m.writeJSON(&sb, 0)
	return []byte(sb.String()), nil
}

// String 渲染为 JSON 文本（2 空格缩进，无尾随换行）
func (m *OrderedMap) String() string {
	var sb strings.Builder
	m.writeJSON(&sb, 0)
	return sb.String()
}

func (m *OrderedMap) writeJSON(sb *strings.Builder, indent int) {
	if m == nil || len(m.keys) == 0 {
		sb.WriteString("{}")
		return
	}

	pad := strings.Repeat("  ", indent)
	childPad := strings.Repeat("  ", indent+1)

	sb.WriteString("{\n")
	for i, key := range m.keys {
		sb.WriteString(childPad)
		sb.WriteString(encodeJSONString(key))
		sb.WriteString(": ")
		switch val := m.vals[key].(type) {
		case *OrderedMap:
			val.writeJSON(sb, indent+1)
		case string:
			sb.WriteString(encodeJSONString(val))
		default:
			sb.WriteString(encodeJSONString(fmt.Sprintf("%v", val)))
		}
		if i < len(m.keys)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(pad)
	sb.WriteString("}")
}

// encodeJSONString 与 JS JSON.stringify 相同的字符串转义规则：
// 仅转义引号、反斜杠与控制字符，非 ASCII 原样输出。
func encodeJSONString(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		case '\b':
			sb.WriteString(`\b`)
		case '\f':
			sb.WriteString(`\f`)
		default:
			if r < 0x20 {
				sb.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	_ = utf8.RuneLen(0)
	return sb.String()
}
