package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/httprule"
)

// TestRenderRoutePath 验证从解析后模板重建的 kratos 路由注册形态:
// 裸变量保持裸形态;带显式 matcher 的变量段重写为 {name:regex}
// (matcher 字面量经 QuoteMeta 转义);单字符变量名与长名一视同仁
// (旧正则实现跳过单字符变量导致其 matcher 保留在路由里)。
func TestRenderRoutePath(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
	}{
		{name: "no vars", template: "/test/noparams", want: "/test/noparams"},
		{name: "plain var stays bare", template: "/test/{message.id}", want: "/test/{message.id}"},
		{name: "single char var matcher rewritten", template: "/test/{a=x}", want: "/test/{a:x}"},
		{name: "literal matcher rewritten", template: "/test/{message.id=test}", want: "/test/{message.id:test}"},
		{name: "wildcard matcher rewritten", template: "/test/{message.name=messages/*}", want: "/test/{message.name:messages/[^/]+}"},
		{name: "matcher with following literal", template: "/test/{message.name=messages/*}/books", want: "/test/{message.name:messages/[^/]+}/books"},
		{name: "plain and explicit vars mixed", template: "/test/{message.id}/{message.name=messages/*}", want: "/test/{message.id}/{message.name:messages/[^/]+}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := httprule.ParseTemplate(tt.template)
			if err != nil {
				t.Fatalf("ParseTemplate(%q): %v", tt.template, err)
			}
			if got := renderRoutePath(tmpl); got != tt.want {
				t.Errorf("renderRoutePath(%q) = %q, want %q", tt.template, got, tt.want)
			}
		})
	}
}

// TestRenderRoutePathRejectsMalformedTemplates 共享解析器对畸形模板
// 报错——旧的自研正则全部照收。
func TestRenderRoutePathRejectsMalformedTemplates(t *testing.T) {
	for _, template := range []string{
		"/test/**/*",
		"/test/{a={b}}",
		"/test/{a}/{a}",
		"",
	} {
		if _, err := httprule.ParseTemplate(template); err == nil {
			t.Errorf("ParseTemplate(%q) should reject the malformed template", template)
		}
	}
}

// TestTemplateVarFieldPaths 变量字段路径按段序确定性返回。
func TestTemplateVarFieldPaths(t *testing.T) {
	tmpl, err := httprule.ParseTemplate("/test/{message.id}/{message.name=messages/*}")
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	got := templateVarFieldPaths(tmpl)
	want := [][]string{{"message", "id"}, {"message", "name"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("templateVarFieldPaths = %v, want %v", got, want)
	}

	tmpl, err = httprule.ParseTemplate("/test/noparams")
	if err != nil {
		t.Fatalf("ParseTemplate: %v", err)
	}
	if got := templateVarFieldPaths(tmpl); got != nil {
		t.Fatalf("templateVarFieldPaths on varless template = %v, want nil", got)
	}
}

func TestPathTemplateRegex(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "single segment",
			value: "messages/*",
			want:  "messages/[^/]+",
		},
		{
			name:  "multi segment",
			value: "messages/**",
			want:  "messages/.*",
		},
		{
			name:  "literal",
			value: "v1.0/*",
			want:  `v1\.0/[^/]+`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathTemplateRegex(tt.value); got != tt.want {
				t.Errorf("expected %s got %s", tt.want, got)
			}
		})
	}
}

func TestFormatStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  string
	}{
		{name: "nil", input: nil, want: ""},
		{name: "empty", input: []string{}, want: ""},
		{name: "single", input: []string{"id"}, want: `[]string{"id"}`},
		{name: "multiple", input: []string{"id", "user.name"}, want: `[]string{"id", "user.name"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatStringSlice(tt.input); got != tt.want {
				t.Errorf("expected %s got %s", tt.want, got)
			}
		})
	}
}

func TestHTTPTemplateBindingAndHandler(t *testing.T) {
	sd := &serviceDesc{
		ServiceType: "Greeter",
		ServiceName: "helloworld.Greeter",
		Methods: []*methodDesc{
			{
				Name:         "SayHello",
				OriginalName: "SayHello",
				Request:      "HelloRequest",
				Reply:        "HelloReply",
				Path:         "/helloworld/{name}",
				Method:       "GET",
				HasVars:      true,
				PathVarsList: `[]string{"name"}`,
			},
			{
				Name:         "CreateHello",
				OriginalName: "CreateHello",
				Request:      "CreateHelloRequest",
				Reply:        "HelloReply",
				Path:         "/helloworld",
				Method:       "POST",
				HasBody:      true,
				BodyField:    "*",
			},
			{
				Name:         "UpdateHello",
				OriginalName: "UpdateHello",
				Request:      "UpdateHelloRequest",
				Reply:        "HelloReply",
				Path:         "/helloworld/{id}",
				Method:       "PATCH",
				HasBody:      true,
				BodyField:    "data",
				HasVars:      true,
				PathVarsList: `[]string{"id"}`,
			},
		},
	}
	got := sd.execute()
	for _, want := range []string{
		// Operation constants
		`const OperationGreeterSayHello = "/helloworld.Greeter/SayHello"`,
		// Interface
		`SayHello(context.Context, *HelloRequest) (*HelloReply, error)`,
		// Register function uses binding.Router
		`func RegisterGreeterHTTPServer(srv binding.Router, svc GreeterHTTPServer) {`,
		`srv.Handle("GET", "/helloworld/{name}", _Greeter_SayHello0_HTTP_Handler(svc))`,
		// Handler uses standard net/http types
		`func _Greeter_SayHello0_HTTP_Handler(svc GreeterHTTPServer) http.HandlerFunc {`,
		`return func(w http.ResponseWriter, r *http.Request) {`,
		// GET handler: query + path binding
		`binding.BindQuery(&in, r.URL.Query())`,
		`binding.BindAllPaths(&in, r, []string{"name"})`,
		// POST handler with body="*"
		`binding.BindBody(r, &in)`,
		// PATCH handler with body="field"
		`binding.BindBodyField(r, &in, "data")`,
		// Response writing
		`binding.WriteResponse(w, r, out)`,
		`binding.WriteError(w, err)`,
		// Business logic call
		`out, err := svc.SayHello(r.Context(), &in)`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("generated template missing %q in:\n%s", want, got)
		}
	}
	// Should NOT contain any framework-specific helper code
	for _, notWant := range []string{
		"http.Context",
		"ctx.Bind",
		"ctx.BindQuery",
		"ctx.BindVars",
		"ctx.Result",
		"ctx.Middleware",
		"http.ServerStream",
		"http.ClientStream",
		"NewWebSocketServerStream",
		"NewServerSentEventServerStream",
		"http.BuildPath",
		"http.Accept",
		"http.ContentType",
	} {
		if strings.Contains(got, notWant) {
			t.Fatalf("generated template should not contain %q:\n%s", notWant, got)
		}
	}
}

func TestHTTPTemplateResponseBody(t *testing.T) {
	sd := &serviceDesc{
		ServiceType: "Greeter",
		ServiceName: "helloworld.Greeter",
		Methods: []*methodDesc{
			{
				Name:         "UploadHello",
				OriginalName: "UploadHello",
				Request:      "UploadHelloRequest",
				Reply:        "UploadHelloReply",
				Path:         "/helloworld/upload",
				Method:       "POST",
				HasBody:      true,
				BodyField:    "*",
				ResponseBody: ".Data",
			},
		},
	}
	got := sd.execute()
	if !strings.Contains(got, `binding.WriteResponse(w, r, out.Data)`) {
		t.Fatalf("generated template should write response body field:\n%s", got)
	}
}

func TestAllFieldsPathBound(t *testing.T) {
	if !allFieldsPathBound([]string{"source", "key"}, [][]string{{"source"}, {"key"}}) {
		t.Fatal("fields fully consumed by path variables must be reported as bound")
	}
	if !allFieldsPathBound(nil, nil) {
		t.Fatal("a request with no fields is vacuously bound")
	}
	if allFieldsPathBound([]string{"source", "filter"}, [][]string{{"source"}, {"key"}}) {
		t.Fatal("a field no path variable binds must not be reported as bound")
	}
	if allFieldsPathBound([]string{"foo"}, [][]string{{"foo", "bar"}}) {
		t.Fatal("a dotted path variable must not count as binding a top-level field")
	}
}
