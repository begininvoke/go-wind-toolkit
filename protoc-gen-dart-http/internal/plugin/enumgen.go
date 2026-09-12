package plugin

import (
	"sort"
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/codegen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type enumGenerator struct {
	pkg  protoreflect.FullName
	enum protoreflect.EnumDescriptor
}

func (e enumGenerator) Generate(f *codegen.File) {
	commentGenerator{descriptor: e.enum}.generateLeading(f, 0)
	enumName := scopedDescriptorTypeName(e.pkg, e.enum)
	if e.enum.Values().Len() == 0 {
		Warn("enum %s has no values; generating fallback dynamic type", e.enum.FullName())
		f.P("// enum ", enumName, " has no values")
		f.P()
		return
	}

	values := make([]protoreflect.EnumValueDescriptor, 0, e.enum.Values().Len())
	rangeEnumValues(e.enum, func(value protoreflect.EnumValueDescriptor, _ bool) {
		values = append(values, value)
	})
	sort.Slice(values, func(i, j int) bool {
		return localeCompare(string(values[i].Name()), string(values[j].Name()))
	})

	// enum declaration
	f.P("enum ", enumName, " {")
	for i, value := range values {
		dartName := dartFieldName(protoEnumToDartName(string(value.Name())))
		protoName := string(value.Name())
		wireNumber := int(value.Number())
		if i == len(values)-1 {
			f.P(t(1), dartName, "(", dartString(protoName), ", ", wireNumber, ");")
		} else {
			f.P(t(1), dartName, "(", dartString(protoName), ", ", wireNumber, "),")
		}
	}
	f.P()
	f.P(t(1), "final String value;")
	f.P(t(1), "/// proto 枚举数值（二进制 wire 编解码用，与 JSON 的 value 字符串互补）")
	f.P(t(1), "final int wire;")
	f.P(t(1), "const ", enumName, "(this.value, this.wire);")
	f.P()
	f.P(t(1), "static ", enumName, " fromString(String v) =>")
	// Escape $ in enumName so Dart does not treat it as string interpolation.
	escapedEnumName := strings.ReplaceAll(enumName, "$", "\\$")
	f.P(t(2), "values.firstWhere((e) => e.value == v, orElse: () => throw ArgumentError('Unknown ", escapedEnumName, " value: ' + v));")
	f.P(t(1), "static ", enumName, " fromWire(int v) =>")
	f.P(t(2), "values.firstWhere((e) => e.wire == v, orElse: () => throw ArgumentError('Unknown ", escapedEnumName, " wire value: ' + v.toString()));")
	f.P(t(1), "@override")
	f.P(t(1), "String toString() => value;")
	f.P("}")
	f.P()
}
