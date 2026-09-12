package plugin

import (
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/codegen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// scopedDescriptorTypeName returns the Dart type name for a descriptor,
// scoped to the given package. Nested types are flattened using '$'
// (matching Dart protobuf convention), and cross-package references are
// prefixed with a PascalCase package prefix.
func scopedDescriptorTypeName(pkg protoreflect.FullName, desc protoreflect.Descriptor) string {
	name := string(desc.Name())
	var prefix string
	if desc.Parent() != desc.ParentFile() {
		prefix = descriptorTypeName(desc.Parent()) + "$"
	}
	if desc.ParentFile().Package() != pkg {
		prefix = packagePrefix(desc.ParentFile().Package()) + prefix
	}
	return prefix + name
}

// descriptorTypeName returns the flattened Dart type name for a descriptor,
// using '$' to separate nested levels (matching Dart protobuf convention).
func descriptorTypeName(desc protoreflect.Descriptor) string {
	name := string(desc.Name())
	var prefix string
	if desc.Parent() != desc.ParentFile() {
		prefix = descriptorTypeName(desc.Parent()) + "$"
	}
	return prefix + name
}

// packagePrefix converts a proto package name to a PascalCase prefix for
// cross-package type references.
// e.g. "einride.example.syntax.v1" -> "EinrideExampleSyntaxV1"
func packagePrefix(pkg protoreflect.FullName) string {
	parts := strings.Split(string(pkg), ".")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

// t returns n levels of indentation for generated code.
func t(n int) string {
	return codegen.Indent(n)
}

// dartEscapeLiteral escapes a path literal for embedding inside a Dart string
// literal: backslash, quote, and dollar. Dart performs $-interpolation in every
// string form, so an unescaped $ from a path literal would either break the
// generated code or silently interpolate an unrelated identifier.
func dartEscapeLiteral(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "$", "\\$")
	return s
}

// dartString wraps s in Dart single quotes, escaping backslashes, single
// quotes, and dollars as needed.
func dartString(s string) string {
	return "'" + dartEscapeLiteral(s) + "'"
}

// protoEnumToDartName converts an UPPER_SNAKE_CASE proto enum value name to
// lowerCamelCase suitable for a Dart enum value.
// e.g. "STATUS_ACTIVE" → "statusActive", "ACTIVE" → "active"
func protoEnumToDartName(s string) string {
	parts := strings.Split(s, "_")
	var sb strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			sb.WriteString(strings.ToLower(p))
		} else {
			if len(p) > 0 {
				sb.WriteString(strings.ToUpper(p[:1]))
				sb.WriteString(strings.ToLower(p[1:]))
			}
		}
	}
	return sb.String()
}

// dartReservedWords is the set of Dart reserved words, built-in identifiers,
// and dart:core type names that must be escaped when used as field names.
// A field named bool/double/int/Map/... is a legal Dart identifier but
// shadows the core type and poisons every later reference to it in the file.
var dartReservedWords = map[string]bool{
	"assert": true, "break": true, "case": true, "catch": true, "class": true,
	"const": true, "continue": true, "default": true, "do": true, "else": true,
	"enum": true, "extends": true, "false": true, "final": true, "finally": true,
	"for": true, "if": true, "in": true, "is": true, "new": true, "null": true,
	"rethrow": true, "return": true, "super": true, "switch": true, "this": true,
	"throw": true, "true": true, "try": true, "var": true, "void": true,
	"while": true, "with": true, "abstract": true, "as": true, "covariant": true,
	"deferred": true, "dynamic": true, "export": true, "extension": true,
	"external": true, "factory": true, "Function": true, "get": true, "hide": true,
	"implements": true, "import": true, "interface": true, "library": true,
	"operator": true, "mixin": true, "part": true, "set": true, "static": true,
	"typedef": true, "late": true, "required": true, "call": true, "await": true,
	"yield": true, "sync": true, "async": true, "show": true,
	// dart:core type names (lowercase builtins and core classes) —
	// shadowing these breaks type references in generated code.
	"bool": true, "double": true, "int": true, "num": true,
	"String": true, "Map": true, "MapEntry": true, "List": true, "Set": true,
	"Null": true, "Future": true, "Stream": true, "Iterator": true,
	"Iterable": true, "Duration": true, "StringBuffer": true, "Symbol": true,
	"Type": true,
}

// dartFieldName returns a safe Dart field name. If name is a Dart reserved
// word, an underscore is appended.
func dartFieldName(name string) string {
	if dartReservedWords[name] {
		return name + "_"
	}
	return name
}

// isNullableDartType returns false for Dart types that are inherently nullable
// (like dynamic) and don't accept the ? suffix.
func isNullableDartType(typeName string) bool {
	return typeName != "dynamic"
}
