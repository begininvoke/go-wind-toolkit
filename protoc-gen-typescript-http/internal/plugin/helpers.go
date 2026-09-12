package plugin

import (
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/codegen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func scopedDescriptorTypeName(pkg protoreflect.FullName, desc protoreflect.Descriptor) string {
	name := string(desc.Name())
	var prefix string
	if desc.Parent() != desc.ParentFile() {
		prefix = descriptorTypeName(desc.Parent()) + "_"
	}
	if desc.ParentFile().Package() != pkg {
		prefix = packagePrefix(desc.ParentFile().Package()) + prefix
	}
	return prefix + name
}

func descriptorTypeName(desc protoreflect.Descriptor) string {
	name := string(desc.Name())
	var prefix string
	if desc.Parent() != desc.ParentFile() {
		prefix = descriptorTypeName(desc.Parent()) + "_"
	}
	return prefix + name
}

func packagePrefix(pkg protoreflect.FullName) string {
	return strings.Join(strings.Split(string(pkg), "."), "") + "_"
}

// t returns n levels of indentation for generated code.
func t(n int) string {
	return codegen.Indent(n)
}

// tsSingleQuote wraps s in TypeScript single quotes, escaping backslashes
// and single quotes as needed. Prettier enforces single quotes for TS strings.
func tsSingleQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return "'" + s + "'"
}
