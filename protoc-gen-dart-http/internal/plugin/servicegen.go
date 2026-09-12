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
	return s.generateClient(f)
}

func (s serviceGenerator) generateClient(f *codegen.File) error {
	clientName := descriptorTypeName(s.service) + "Client"
	commentGenerator{descriptor: s.service}.generateLeading(f, 0)
	f.P("class ", clientName, " {")
	f.P(t(1), "final ClientTransport _transport;")
	f.P()
	f.P(t(1), clientName, "(this._transport);")
	f.P()

	var methodErrs []error
	first := true
	protowalk.RangeMethods(s.service.Methods(), func(method protoreflect.MethodDescriptor) {
		ok, reason := httprule.SupportedMethod(method)
		if !ok {
			Warn("method %s.%s skipped in client: %s", s.service.FullName(), method.Name(), reason)
			return
		}
		if !first {
			f.P()
		}
		first = false
		if err := s.generateMethod(f, method); err != nil {
			methodErrs = append(methodErrs, fmt.Errorf("generate method %s.%s: %w", s.service.FullName(), method.Name(), err))
		}
	})
	if len(methodErrs) > 0 {
		return fmt.Errorf("%d method(s) failed: %w", len(methodErrs), errors.Join(methodErrs...))
	}
	f.P("}")
	f.P()
	return nil
}

func (s serviceGenerator) generateMethod(f *codegen.File, method protoreflect.MethodDescriptor) error {
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

	// 每个 additional_bindings 条目生成一个 <method>Alt<N> 变体方法。
	baseName := codegen.LowerFirst(string(method.Name()))
	if err := s.generateUnaryMethod(f, method, rule, baseName, 0); err != nil {
		return err
	}
	for i, additional := range rule.AdditionalRules {
		f.P()
		if err := s.generateUnaryMethod(f, method, additional, fmt.Sprintf("%sAlt%d", baseName, i+1), i+1); err != nil {
			return fmt.Errorf("generate additional binding %d: %w", i+1, err)
		}
	}
	return nil
}

// generateUnaryMethod 为一个 HTTP 绑定生成一个 Dart 客户端方法。
// bindingIndex 0 为注解里的主绑定;>0 表示 additional_bindings 变体,
// 文档注释标注其路由。
func (s serviceGenerator) generateUnaryMethod(
	f *codegen.File,
	method protoreflect.MethodDescriptor,
	rule httprule.Rule,
	dartMethodName string,
	bindingIndex int,
) error {
	if bindingIndex == 0 {
		commentGenerator{descriptor: method}.generateLeading(f, 1)
	} else {
		f.P(t(1), "/// Alternate HTTP binding #", bindingIndex, " of ", codegen.LowerFirst(string(method.Name())), ": ", rule.Method, " ", rule.Template.String())
	}
	outputType := typeFromMessage(s.pkg, method.Output())
	inputType := typeFromMessage(s.pkg, method.Input())

	usesRequest := httprule.MethodUsesRequest(rule, method.Input(), IsWellKnownType)
	paramName := "request"
	if !usesRequest {
		paramName = "_request"
	}

	f.P(t(1), "Future<", outputType.Reference(), "> ", dartMethodName, "(", inputType.Reference(), " ", paramName, ", {Map<String, String>? headers}) async {")
	generateMethodPathValidation(f, method.Input(), rule)
	generateMethodPath(f, method.Input(), rule)
	bodyVar := generateMethodBody(f, method.Input(), rule)
	hasQP := generateMethodQuery(f, method.Input(), rule)
	uriVar := "path"
	if hasQP {
		f.P(t(2), "var uri = path;")
		f.P(t(2), "if (queryParams.isNotEmpty) {")
		f.P(t(3), "uri += '?${queryParams.join(\"&\")}';")
		f.P(t(2), "}")
		uriVar = "uri"
	}
	f.P(t(2), "final result = await _transport.unary(", uriVar, ", ", dartString(rule.Method), ", ", bodyVar, ", TransportMeta(")
	f.P(t(3), "service: ", dartString(string(method.Parent().Name())), ",")
	f.P(t(3), "method: ", dartString(string(method.Name())), ",")
	f.P(t(2), "), headers: headers);")
	f.P(t(2), "return ", returnTypeExpr(outputType, method.Output()), ";")
	f.P(t(1), "}")
	return nil
}

// returnTypeExpr generates the expression to convert a transport result to the output type.
func returnTypeExpr(outputType Type, output protoreflect.MessageDescriptor) string {
	if _, isWKT := WellKnownType(output); isWKT {
		return "result as " + outputType.Name
	}
	return outputType.Name + ".fromJson(result as Map<String, dynamic>)"
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
		dartPath := dartAccessPath(fp, input)
		protoPath := strings.Join(fp, ".")
		errMsg := "missing required field request." + protoPath
		f.P(t(2), "if (request.", dartPath, " == null) {")
		f.P(t(3), "throw ArgumentError(", dartString(errMsg), ");")
		f.P(t(2), "}")
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
			fieldPath := dartAccessPath(seg.Variable.FieldPath, input)
			pathParts = append(pathParts, "${request."+fieldPath+"}")
		case httprule.SegmentKindLiteral:
			pathParts = append(pathParts, dartEscapeLiteral(seg.Literal))
		case httprule.SegmentKindMatchSingle:
			pathParts = append(pathParts, "*")
		case httprule.SegmentKindMatchMultiple:
			pathParts = append(pathParts, "**")
		}
	}
	path := "/" + strings.Join(pathParts, "/")
	if rule.Template.Verb != "" {
		path += ":" + dartEscapeLiteral(rule.Template.Verb)
	}
	f.P(t(2), "final path = '", path, "';")
}

func generateMethodBody(
	f *codegen.File,
	input protoreflect.MessageDescriptor,
	rule httprule.Rule,
) string {
	isWKT := IsWellKnownType(input)
	switch {
	case rule.Body == "":
		return "null"
	case rule.Body == "*":
		if isWKT {
			f.P(t(2), "final body = jsonEncode(request ?? {});")
		} else if pathVars := rule.Template.PathVariableFieldPaths(); len(pathVars) > 0 {
			// Path-bound fields must not be duplicated in the body: the server
			// binds them from the URL path, and a field arriving from both
			// sources conflicts on its (synthetic) oneof.
			f.P(t(2), "final bodyMap = request.toJson();")
			for _, fp := range pathVars {
				f.P(t(2), jsonMapRemoveExpr("bodyMap", rawJsonPathSegments(fp, input)))
			}
			f.P(t(2), "final body = jsonEncode(bodyMap);")
		} else {
			f.P(t(2), "final body = jsonEncode(request.toJson());")
		}
		return "body"
	default:
		if isWKT {
			// WKT inputs don't have named body fields; fall back to encoding the whole request.
			f.P(t(2), "final body = jsonEncode(request ?? {});")
			return "body"
		}
		bodyField := input.Fields().ByName(protoreflect.Name(rule.Body))
		if bodyField == nil {
			Warn("body field %q referenced in http rule not found in message %s; falling back to full request", rule.Body, input.FullName())
			f.P(t(2), "final body = jsonEncode(request.toJson());")
			return "body"
		}
		dartName := dartFieldName(bodyField.JSONName())
		f.P(t(2), "final body = jsonEncode(request.", dartName, "?.toJson() ?? {});")
		return "body"
	}
}

// jsonMapRemoveExpr renders a Dart expression removing a (possibly nested)
// key from the body map produced by toJson, e.g.
//
//	bodyMap.remove('channelId')
//	bodyMap['friend']?.remove('id')
func jsonMapRemoveExpr(root string, namePath []string) string {
	var b strings.Builder
	b.WriteString(root)
	for _, mid := range namePath[:len(namePath)-1] {
		b.WriteString("['")
		b.WriteString(mid)
		b.WriteString("']?")
	}
	b.WriteString(".remove('")
	b.WriteString(namePath[len(namePath)-1])
	b.WriteString("');")
	return b.String()
}

func generateMethodQuery(
	f *codegen.File,
	input protoreflect.MessageDescriptor,
	rule httprule.Rule,
) bool {
	if !httprule.HasQueryParams(input, rule, IsWellKnownType) {
		return false
	}
	f.P(t(2), "final queryParams = <String>[];")
	httprule.WalkJSONLeafFields(input, IsWellKnownType, func(path httprule.FieldPath, field protoreflect.FieldDescriptor) {
		if rule.QueryExcluded(path) {
			return
		}
		if protowalk.IsMessageCollectionField(field) {
			return
		}
		jp := jsonAccessPath(path, input)
		jpNonNull := jsonNonNullAccessPath(path, input)
		jsonName := jsonNameFromPath(path, input)
		f.P(t(2), "if (request.", jp, " != null) {")
		switch {
		case field.IsMap():
			f.P(t(3), "request.", jpNonNull, "!.forEach((key, value) {")
			f.P(t(4), "queryParams.add('", jsonName, "[key]=${Uri.encodeComponent(value.toString())}');")
			f.P(t(3), "});")
		case field.IsList():
			f.P(t(3), "request.", jpNonNull, "!.forEach((x) {")
			f.P(t(4), "queryParams.add('", jsonName, "=${Uri.encodeComponent(x.toString())}');")
			f.P(t(3), "});")
		default:
			f.P(t(3), "queryParams.add('", jsonName, "=${Uri.encodeComponent(request.", jpNonNull, "!.toString())}');")
		}
		f.P(t(2), "}")
	})
	return true
}

// dartAccessPath returns the Dart field access path for a field path.
// Uses ?. for intermediate segments to handle null safety.
// e.g. ["shipper", "name"] → "shipper?.name"
func dartAccessPath(path httprule.FieldPath, message protoreflect.MessageDescriptor) string {
	segs := dartPathSegments(path, message)
	return strings.Join(segs, "?.")
}

// jsonAccessPath returns the Dart access path using JSON names.
func jsonAccessPath(path httprule.FieldPath, message protoreflect.MessageDescriptor) string {
	segs := jsonPathSegments(path, message)
	return strings.Join(segs, "?.")
}

// jsonNonNullAccessPath returns the Dart access path with non-null assertions
// (!.) for intermediate segments. Used inside if-null blocks where we've
// already verified the value is non-null.
// e.g. ["filterExpr", "type"] → "filterExpr!.type"
func jsonNonNullAccessPath(path httprule.FieldPath, message protoreflect.MessageDescriptor) string {
	segs := jsonPathSegments(path, message)
	return strings.Join(segs, "!.")
}

// jsonNameFromPath returns the raw JSON field name path for URL query parameter keys.
// Uses raw JSON names without Dart escaping.
func jsonNameFromPath(path httprule.FieldPath, message protoreflect.MessageDescriptor) string {
	if len(path) == 0 {
		return ""
	}
	segs := rawJsonPathSegments(path, message)
	return strings.Join(segs, ".")
}

// dartPathSegments returns path segments using proto JSON names with Dart reserved
// word escaping applied. Used for Dart field access (e.g. request.fieldName).
func dartPathSegments(path httprule.FieldPath, message protoreflect.MessageDescriptor) []string {
	segs := make([]string, len(path))
	for i, p := range path {
		field := message.Fields().ByName(protoreflect.Name(p))
		if field == nil {
			Warn("field %q not found in message %s; path segment may be incorrect", p, message.FullName())
			segs[i] = dartFieldName(p)
			continue
		}
		segs[i] = dartFieldName(field.JSONName())
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

// rawJsonPathSegments returns raw JSON field names without Dart escaping.
// Used for URL query parameter keys.
func rawJsonPathSegments(path httprule.FieldPath, message protoreflect.MessageDescriptor) []string {
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
				break
			}
			nested := field.Message()
			if nested == nil {
				break
			}
			message = nested
		}
	}
	return segs
}

func jsonPathSegments(path httprule.FieldPath, message protoreflect.MessageDescriptor) []string {
	return dartPathSegments(path, message)
}
