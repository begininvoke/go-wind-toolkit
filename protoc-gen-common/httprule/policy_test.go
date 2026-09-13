package httprule_test

import (
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/httprule"
)

func mustParse(t *testing.T, s string) httprule.Template {
	t.Helper()
	tmpl, err := httprule.ParseTemplate(s)
	if err != nil {
		t.Fatalf("ParseTemplate(%q): %v", s, err)
	}
	return tmpl
}

// policyFixture 合成含嵌套消息、WKT 形态消息(本地 Money 代替真实 WKT)、
// 集合字段、http 规则注解与 default_host 的描述符,供绑定策略测试使用。
func policyFixture(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()

	str := func(s string) *string { return proto.String(s) }
	i32 := func(n int32) *int32 { return proto.Int32(n) }
	opt := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	rep := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	msg := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	str32 := descriptorpb.FieldDescriptorProto_TYPE_STRING
	field := func(name string, num int32, typ descriptorpb.FieldDescriptorProto_Type, label descriptorpb.FieldDescriptorProto_Label, typeName string) *descriptorpb.FieldDescriptorProto {
		f := &descriptorpb.FieldDescriptorProto{
			Name:   str(name),
			Number: i32(num),
			Label:  label.Enum(),
			Type:   typ.Enum(),
		}
		if typeName != "" {
			f.TypeName = str(typeName)
		}
		return f
	}
	getRule := func(path string) *descriptorpb.MethodOptions {
		opts := &descriptorpb.MethodOptions{}
		proto.SetExtension(opts, annotations.E_Http, &annotations.HttpRule{
			Pattern: &annotations.HttpRule_Get{Get: path},
		})
		return opts
	}
	postRule := func(path, body string) *descriptorpb.MethodOptions {
		opts := &descriptorpb.MethodOptions{}
		proto.SetExtension(opts, annotations.E_Http, &annotations.HttpRule{
			Pattern: &annotations.HttpRule_Post{Post: path},
			Body:    body,
		})
		return opts
	}
	withDefaultHost := func(host string) *descriptorpb.ServiceOptions {
		opts := &descriptorpb.ServiceOptions{}
		proto.SetExtension(opts, annotations.E_DefaultHost, host)
		return opts
	}

	fdp := &descriptorpb.FileDescriptorProto{
		Name:    str("policytest.proto"),
		Package: str("policytest"),
		Syntax:  str("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: str("Req"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("q", 1, str32, opt, ""),
					field("money", 2, msg, opt, ".policytest.Money"),
					field("labels", 3, str32, rep, ""),
					field("owner", 4, msg, opt, ".policytest.Owner"),
					field("filter", 5, msg, opt, ".policytest.Filter"),
				},
			},
			{
				Name:  str("Owner"),
				Field: []*descriptorpb.FieldDescriptorProto{field("name", 1, str32, opt, "")},
			},
			{
				Name:  str("Filter"),
				Field: []*descriptorpb.FieldDescriptorProto{field("keyword", 1, str32, opt, "")},
			},
			{Name: str("Money")},
			{
				Name:  str("OnlyQ"),
				Field: []*descriptorpb.FieldDescriptorProto{field("q", 1, str32, opt, "")},
			},
			{
				Name:  str("CollOnly"),
				Field: []*descriptorpb.FieldDescriptorProto{field("labels", 1, str32, rep, "")},
			},
			{
				Name:  str("CollMsgOnly"),
				Field: []*descriptorpb.FieldDescriptorProto{field("owners", 1, msg, rep, ".policytest.Owner")},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name:    str("Svc"),
				Options: withDefaultHost("api.example.com"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: str("GetQ"), InputType: str(".policytest.Req"), OutputType: str(".policytest.Req"), Options: getRule("/v1/req")},
					{Name: str("PostQ"), InputType: str(".policytest.Req"), OutputType: str(".policytest.Req"), Options: postRule("/v1/req", "*")},
					{
						Name:            str("ClientStream"),
						InputType:       str(".policytest.Req"),
						OutputType:      str(".policytest.Req"),
						ClientStreaming: proto.Bool(true),
						Options:         getRule("/v1/req:stream"),
					},
					{Name: str("NoRule"), InputType: str(".policytest.Req"), OutputType: str(".policytest.Req")},
				},
			},
			{
				Name:    str("Svc2"),
				Options: withDefaultHost("other.example.com"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: str("GetOnlyQ"), InputType: str(".policytest.OnlyQ"), OutputType: str(".policytest.OnlyQ"), Options: getRule("/v1/{q}")},
				},
			},
		},
	}

	fd, err := protodesc.NewFile(fdp, nil)
	if err != nil {
		t.Fatalf("protodesc.NewFile: %v", err)
	}
	return fd
}

// moneyAwareWKT 把本地 Money 消息认作 WKT 叶子,模拟两侧生成器对真实
// WKT(google.protobuf.*)的叶子截断策略。
func moneyAwareWKT(desc protoreflect.Descriptor) bool {
	return desc != nil && string(desc.FullName()) == "policytest.Money"
}

func reqDesc(fd protoreflect.FileDescriptor) protoreflect.MessageDescriptor {
	return fd.Messages().ByName("Req")
}

func TestWalkJSONLeafFields(t *testing.T) {
	fd := policyFixture(t)

	var got []string
	httprule.WalkJSONLeafFields(reqDesc(fd), moneyAwareWKT, func(path httprule.FieldPath, field protoreflect.FieldDescriptor) {
		got = append(got, path.String()+"("+field.JSONName()+")")
	})

	// 字段声明序;q 与 Money 停在叶子,labels 是集合字段但也是叶(集合
	// 过滤在 HasQueryParams 里做),owner/filter 递归。
	want := []string{"q(q)", "money(money)", "labels(labels)", "owner.name(name)", "filter.keyword(keyword)"}
	assert.DeepEqual(t, want, got)
}

func TestHasQueryParams(t *testing.T) {
	fd := policyFixture(t)

	for _, tt := range []struct {
		name string
		rule httprule.Rule
		msg  protoreflect.MessageDescriptor
		want bool
	}{
		{
			name: "plain get has query params",
			rule: httprule.Rule{Method: "GET", Template: mustParse(t, "/v1/req")},
			msg:  reqDesc(fd),
			want: true,
		},
		{
			name: "wildcard body leaves none",
			rule: httprule.Rule{Method: "POST", Template: mustParse(t, "/v1/req"), Body: "*"},
			msg:  reqDesc(fd),
			want: false,
		},
		{
			name: "named body still leaves other leaves",
			rule: httprule.Rule{Method: "POST", Template: mustParse(t, "/v1/req"), Body: "filter"},
			msg:  reqDesc(fd),
			want: true,
		},
		{
			name: "fully path-bound message has none",
			rule: httprule.Rule{Method: "GET", Template: mustParse(t, "/v1/req/{q}")},
			msg:  fd.Messages().ByName("OnlyQ"),
			want: false,
		},
		{
			// 重复标量字段是合法查询参数,不算集合。
			name: "scalar collection still counts",
			rule: httprule.Rule{Method: "GET", Template: mustParse(t, "/v1/coll")},
			msg:  fd.Messages().ByName("CollOnly"),
			want: true,
		},
		{
			// 仅含消息型集合字段时没有任何可作查询参数的叶子。
			name: "message-collection-only message has none",
			rule: httprule.Rule{Method: "GET", Template: mustParse(t, "/v1/coll")},
			msg:  fd.Messages().ByName("CollMsgOnly"),
			want: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, httprule.HasQueryParams(tt.msg, tt.rule, moneyAwareWKT))
		})
	}
}

func TestMethodUsesRequest(t *testing.T) {
	fd := policyFixture(t)

	pathVar := httprule.Rule{Method: "GET", Template: mustParse(t, "/v1/req/{q}")}
	assert.Assert(t, httprule.MethodUsesRequest(pathVar, fd.Messages().ByName("OnlyQ"), moneyAwareWKT))

	body := httprule.Rule{Method: "POST", Template: mustParse(t, "/v1/req"), Body: "*"}
	assert.Assert(t, httprule.MethodUsesRequest(body, reqDesc(fd), moneyAwareWKT))

	query := httprule.Rule{Method: "GET", Template: mustParse(t, "/v1/req")}
	assert.Assert(t, httprule.MethodUsesRequest(query, reqDesc(fd), moneyAwareWKT))

	none := httprule.Rule{Method: "GET", Template: mustParse(t, "/v1/coll")}
	assert.Assert(t, !httprule.MethodUsesRequest(none, fd.Messages().ByName("CollMsgOnly"), moneyAwareWKT))
}

func TestSupportedMethod(t *testing.T) {
	fd := policyFixture(t)
	methods := fd.Services().Get(0).Methods()

	ok, reason := httprule.SupportedMethod(methods.ByName("GetQ"))
	assert.Assert(t, ok, "annotated unary method must be supported; reason: %s", reason)
	assert.Equal(t, "", reason)

	ok, reason = httprule.SupportedMethod(methods.ByName("NoRule"))
	assert.Assert(t, !ok)
	assert.Equal(t, "no http rule annotation (google.api.http)", reason)

	ok, reason = httprule.SupportedMethod(methods.ByName("ClientStream"))
	assert.Assert(t, !ok)
	assert.Equal(t, "client-only streaming is not supported", reason)
}

func TestDefaultHost(t *testing.T) {
	fd := policyFixture(t)
	services := fd.Services()

	assert.Equal(t, "api.example.com", httprule.DefaultHost(services.Get(0)))
	assert.Equal(t, "other.example.com", httprule.DefaultHost(services.Get(1)))
}

func TestFirstDefaultHost(t *testing.T) {
	fd := policyFixture(t)

	var warns []string
	host := httprule.FirstDefaultHost([]protoreflect.FileDescriptor{fd}, func(format string, args ...interface{}) {
		warns = append(warns, fmt.Sprintf(format, args...))
	})
	assert.Equal(t, "api.example.com", host, "the first service's default_host must win")
	assert.Equal(t, 1, len(warns), "conflicting host must be reported once; got %v", warns)
	assert.Assert(t, strings.Contains(warns[0], "policytest.Svc2"), "warning must name the conflicting service; got %q", warns[0])
}
