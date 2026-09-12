package plugin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/codegen"
	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/httprule"
	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/protowalk"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type serviceGenerator struct {
	pkg     protoreflect.FullName
	service protoreflect.ServiceDescriptor
}

func (s serviceGenerator) Generate(f *codegen.File) error {
	s.generateInterface(f)
	return s.generateClient(f)
}

func (s serviceGenerator) generateInterface(f *codegen.File) {
	commentGenerator{descriptor: s.service}.generateLeading(f, 0)
	f.P("export interface ", descriptorTypeName(s.service), " {")
	protowalk.RangeMethods(s.service.Methods(), func(method protoreflect.MethodDescriptor) {
		ok, reason := httprule.SupportedMethod(method)
		if !ok {
			Warn("method %s.%s skipped: %s", s.service.FullName(), method.Name(), reason)
			return
		}
		if protowalk.IsStreamingMethod(method) {
			r, ok := httprule.Get(method)
			if !ok {
				Warn("streaming method %s.%s has no http rule; skipping", s.service.FullName(), method.Name())
				return
			}
			rule, err := httprule.ParseRule(r)
			if err != nil {
				Warn("streaming method %s.%s has invalid http rule: %v; skipping", s.service.FullName(), method.Name(), err)
				return
			}
			generateStreamInterfaceMethod(f, s.pkg, method, rule)
			for i := range rule.AdditionalRules {
				Warn("method %s.%s: streaming additional binding %d skipped (streaming methods support the primary binding only)", s.service.FullName(), method.Name(), i+1)
			}
			return
		}
		commentGenerator{descriptor: method}.generateLeading(f, 1)
		input := typeFromMessage(s.pkg, method.Input())
		output := typeFromMessage(s.pkg, method.Output())
		f.P(t(1), method.Name(), "(")
		f.P(t(2), "request: ", input.Reference(), ",")
		f.P(t(1), "): Promise<", output.Reference(), ">;")
		s.generateAltInterfaceMethods(f, method, input, output)
	})
	f.P("}")
	f.P()
}

// generateAltInterfaceMethods declares one interface method per
// additional_bindings entry, named <Method>Alt<N>.
func (s serviceGenerator) generateAltInterfaceMethods(
	f *codegen.File,
	method protoreflect.MethodDescriptor,
	input, output Type,
) {
	r, ok := httprule.Get(method)
	if !ok {
		return
	}
	rule, err := httprule.ParseRule(r)
	if err != nil {
		Warn("method %s.%s has invalid http rule: %v; additional bindings not declared", s.service.FullName(), method.Name(), err)
		return
	}
	for i, additional := range rule.AdditionalRules {
		f.P(t(1), "/** Alternate HTTP binding #", i+1, " of ", method.Name(), ": ", additional.Method, " ", additional.Template.String(), " */")
		f.P(t(1), method.Name(), "Alt", i+1, "(")
		f.P(t(2), "request: ", input.Reference(), ",")
		f.P(t(1), "): Promise<", output.Reference(), ">;")
	}
}

func (s serviceGenerator) generateClient(f *codegen.File) error {
	f.P(
		"export function create",
		descriptorTypeName(s.service),
		"Client(",
		"\n",
		t(1),
		"transport: ClientTransport,",
		"\n",
		"): ",
		descriptorTypeName(s.service),
		" {",
	)
	f.P(t(1), "return {")
	var methodErrs []error
	protowalk.RangeMethods(s.service.Methods(), func(method protoreflect.MethodDescriptor) {
		ok, reason := httprule.SupportedMethod(method)
		if !ok {
			Warn("method %s.%s skipped in client: %s", s.service.FullName(), method.Name(), reason)
			return
		}
		if err := s.generateMethod(f, method); err != nil {
			methodErrs = append(methodErrs, fmt.Errorf("generate method %s.%s: %w", s.service.FullName(), method.Name(), err))
		}
	})
	if len(methodErrs) > 0 {
		return fmt.Errorf("%d method(s) failed: %w", len(methodErrs), errors.Join(methodErrs...))
	}
	f.P(t(1), "};")
	f.P("}")
	return nil
}

func (s serviceGenerator) generateMethod(f *codegen.File, method protoreflect.MethodDescriptor) error {
	outputType := typeFromMessage(s.pkg, method.Output())
	r, ok := httprule.Get(method)
	if !ok {
		Warn("method %s.%s has no http rule annotation; skipping", s.service.FullName(), method.Name())
		return nil
	}
	rule, err := httprule.ParseRule(r)
	if err != nil {
		return fmt.Errorf("parse http rule: %w", err)
	}
	if protowalk.IsStreamingMethod(method) {
		generateStreamClientMethod(f, s.pkg, method, rule)
		for i := range rule.AdditionalRules {
			Warn("method %s.%s: streaming additional binding %d skipped (streaming methods support the primary binding only)", s.service.FullName(), method.Name(), i)
		}
		return nil
	}

	// 每个 additional_bindings 条目生成一个 <Method>Alt<N> 成员。
	if err := s.generateUnaryMethod(f, method, rule, string(method.Name()), outputType, 0); err != nil {
		return err
	}
	for i, additional := range rule.AdditionalRules {
		f.P()
		if err := s.generateUnaryMethod(f, method, additional, fmt.Sprintf("%sAlt%d", method.Name(), i+1), outputType, i+1); err != nil {
			return fmt.Errorf("generate additional binding %d: %w", i+1, err)
		}
	}
	return nil
}

// generateUnaryMethod generates the client object member for one HTTP binding.
// bindingIndex 0 is the primary binding; >0 marks an additional_bindings
// variant whose comment advertises its route.
func (s serviceGenerator) generateUnaryMethod(
	f *codegen.File,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
	tsMethodName string,
	outputType Type,
	bindingIndex int,
) error {
	if bindingIndex > 0 {
		f.P(t(2), "// Alternate HTTP binding #", bindingIndex, " of ", method.Name(), ": ", rule.Method, " ", rule.Template.String())
	}
	paramName := "request"
	if !httprule.MethodUsesRequest(rule, method.Input(), IsWellKnownType) {
		paramName = "_request"
	}
	f.P(t(2), tsMethodName, "(", paramName, ") {")
	generateMethodPathValidation(f, method.Input(), rule)
	generateMethodPath(f, method.Input(), rule)
	generateMethodBody(f, method.Input(), rule)
	hasQP := generateMethodQuery(f, method.Input(), rule)
	uriVar := "path"
	if hasQP {
		f.P(t(3), "let uri = path;")
		f.P(t(3), "if (queryParams.length > 0) {")
		f.P(t(4), "uri += `?${queryParams.join('&')}`;")
		f.P(t(3), "}")
		uriVar = "uri"
	}
	f.P(t(3), "return transport.unary(", uriVar, ", ", tsSingleQuote(rule.Method), ", body, {")
	f.P(t(4), "service: '", method.Parent().Name(), "',")
	f.P(t(4), "method: '", method.Name(), "',")
	f.P(t(3), "}) as Promise<", outputType.Reference(), ">;")
	f.P(t(2), "},")
	return nil
}

func generateMethodPathValidation(
	f *codegen.File,
	input protoreflect.MessageDescriptor,
	rule httprule.Rule,
) {
	for _, seg := range rule.Template.Segments {
		if seg.Kind != httprule.SegmentKindVariable {
			continue
		}
		fp := seg.Variable.FieldPath
		nullPath := nullPropagationPath(fp, input)
		protoPath := strings.Join(fp, ".")
		errMsg := "missing required field request." + protoPath
		f.P(t(3), "if (request.", nullPath, " === undefined || request.", nullPath, " === null) {")
		f.P(t(4), "throw new Error(", tsSingleQuote(errMsg), ");")
		f.P(t(3), "}")
	}
}

func generateMethodPath(
	f *codegen.File,
	input protoreflect.MessageDescriptor,
	rule httprule.Rule,
) {
	pathParts := make([]string, 0, len(rule.Template.Segments))
	for _, seg := range rule.Template.Segments {
		switch seg.Kind {
		case httprule.SegmentKindVariable:
			fieldPath := jsonPath(seg.Variable.FieldPath, input)
			pathParts = append(pathParts, "${request."+fieldPath+"}")
		case httprule.SegmentKindLiteral:
			pathParts = append(pathParts, escapeTemplateLiteral(seg.Literal))
		case httprule.SegmentKindMatchSingle:
			pathParts = append(pathParts, "*")
		case httprule.SegmentKindMatchMultiple:
			pathParts = append(pathParts, "**")
		}
	}
	path := "/" + strings.Join(pathParts, "/")
	if rule.Template.Verb != "" {
		path += ":" + escapeTemplateLiteral(rule.Template.Verb)
	}
	f.P(t(3), "const path = `", path, "`;")
}

// escapeTemplateLiteral escapes characters that have special meaning inside a
// JavaScript template literal (backtick string) to prevent generated code
// injection and syntax errors.
func escapeTemplateLiteral(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "${", "\\${")
	return s
}

func generateMethodBody(
	f *codegen.File,
	input protoreflect.MessageDescriptor,
	rule httprule.Rule,
) {
	switch {
	case rule.Body == "":
		f.P(t(3), "const body = null;")
	case rule.Body == "*":
		if pathVars := rule.Template.PathVariableFieldPaths(); len(pathVars) > 0 {
			// Path-bound fields must not be duplicated in the body: the server
			// binds them from the URL path, and a field arriving from both
			// sources conflicts on its (synthetic) oneof.
			f.P(t(3), "const bodyMap = { ...request } as Record<string, unknown>;")
			for _, fp := range pathVars {
				tsBodyStripStmts(f, jsonPathSegments(fp, input))
			}
			f.P(t(3), "const body = JSON.stringify(bodyMap);")
		} else {
			f.P(t(3), "const body = JSON.stringify(request);")
		}
	default:
		bodyField := input.Fields().ByName(protoreflect.Name(rule.Body))
		if bodyField == nil {
			Warn("body field %q referenced in http rule not found in message %s; falling back to full request", rule.Body, input.FullName())
			f.P(t(3), "const body = JSON.stringify(request);")
			return
		}
		nullPath := nullPropagationPath(httprule.FieldPath{rule.Body}, input)
		f.P(t(3), "const body = JSON.stringify(request?.", nullPath, " ?? {});")
	}
}

// tsBodyStripStmts emits TS statements removing a (possibly nested) key from
// the shallow-copied body map. Intermediate levels are re-copied first so the
// delete never mutates the caller's request object.
func tsBodyStripStmts(f *codegen.File, namePath []string) {
	for i := 1; i < len(namePath); i++ {
		prefix := strings.Join(namePath[:i], `"]["`)
		f.P(t(3), `bodyMap["`+prefix+`"] = { ...bodyMap["`+prefix+`"] };`)
	}
	full := strings.Join(namePath, `"]["`)
	f.P(t(3), `delete bodyMap["`+full+`"];`)
}

func generateMethodQuery(
	f *codegen.File,
	input protoreflect.MessageDescriptor,
	rule httprule.Rule,
) bool {
	if !httprule.HasQueryParams(input, rule, IsWellKnownType) {
		return false
	}
	f.P(t(3), "const queryParams: string[] = [];")
	httprule.WalkJSONLeafFields(input, IsWellKnownType, func(path httprule.FieldPath, field protoreflect.FieldDescriptor) {
		if rule.QueryExcluded(path) {
			return
		}
		if protowalk.IsMessageCollectionField(field) {
			return
		}
		nullPath := nullPropagationPath(path, input)
		jp := jsonPath(path, input)
		f.P(t(3), "if (request.", nullPath, ") {")
		switch {
		case field.IsMap():
			f.P(t(4), "Object.entries(request.", jp, ").forEach(([key, value]) => {")
			f.P(t(5), "queryParams.push(")
			f.P(t(6), "`", jp, "[key]=${encodeURIComponent(value.toString())}`,")
			f.P(t(5), ");")
			f.P(t(4), "});")
		case field.IsList():
			f.P(t(4), "request.", jp, ".forEach((x) => {")
			f.P(t(5), "queryParams.push(")
			f.P(t(6), "`", jp, "=${encodeURIComponent(x.toString())}`,")
			f.P(t(5), ");")
			f.P(t(4), "});")
		default:
			f.P(t(4), "queryParams.push(")
			f.P(t(5), "`", jp, "=${encodeURIComponent(request.", jp, ".toString())}`,")
			f.P(t(4), ");")
		}
		f.P(t(3), "}")
	})
	return true
}

func jsonPath(path httprule.FieldPath, message protoreflect.MessageDescriptor) string {
	return strings.Join(jsonPathSegments(path, message), ".")
}

func nullPropagationPath(path httprule.FieldPath, message protoreflect.MessageDescriptor) string {
	return strings.Join(jsonPathSegments(path, message), "?.")
}

func jsonPathSegments(path httprule.FieldPath, message protoreflect.MessageDescriptor) []string {
	segs := make([]string, len(path))
	for i, p := range path {
		field := message.Fields().ByName(protoreflect.Name(p))
		if field == nil {
			Warn("field %q not found in message %s; path segment may be incorrect", p, message.FullName())
			segs[i] = p
			continue
		}
		segs[i] = field.JSONName()
		if i < len(path)-1 {
			if field.Kind() != protoreflect.MessageKind {
				Warn("field %q in message %s is not a message type; cannot traverse nested path %s", p, message.FullName(), path.String())
				break
			}
			nested := field.Message()
			if nested == nil {
				Warn("field %q in message %s has no valid message descriptor; cannot traverse nested path", p, message.FullName())
				break
			}
			message = nested
		}
	}
	return segs
}
