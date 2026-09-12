package httprule

import (
	"testing"

	"gotest.tools/v3/assert"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// fakeMethod 仅实现 Options(),其余 MethodDescriptor 方法经接口内嵌
// 保持未实现——Get 只调用 Options()。
type fakeMethod struct {
	protoreflect.MethodDescriptor
	options proto.Message
}

func (m *fakeMethod) Options() proto.Message {
	return m.options
}

func Test_Get(t *testing.T) {
	t.Parallel()

	// 无注解:默认 MethodOptions 上取不到 E_Http。
	bare := &fakeMethod{options: &descriptorpb.MethodOptions{}}
	if r, ok := Get(bare); ok || r != nil {
		t.Errorf("method without annotation: Get = (%v, %v), want (nil, false)", r, ok)
	}

	// 有注解:经 SetExtension 设置后原样取回。
	opts := &descriptorpb.MethodOptions{}
	proto.SetExtension(opts, annotations.E_Http, &annotations.HttpRule{
		Pattern: &annotations.HttpRule_Get{Get: "/v1/x"},
	})
	annotated := &fakeMethod{options: opts}
	r, ok := Get(annotated)
	assert.Assert(t, ok, "annotated method must yield its rule")
	if r != nil {
		assert.Equal(t, "/v1/x", r.GetGet())
	}
}

// Test_ParseRule 表驱动覆盖各 pattern 类型的方法与模板提取、body 透传、
// additional_bindings 递归解析,以及缺 pattern / 畸形模板 / 畸形嵌套绑定
// 的错误路径。
func Test_ParseRule(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		rule *annotations.HttpRule
		// want 主绑定的期望形态;wantErr 非空时期望整体报错。
		wantMethod   string
		wantSegments int
		wantVerb     string
		wantBody     string
		wantErr      string
	}{
		{
			name:         "get",
			rule:         &annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: "/v1/x"}},
			wantMethod:   "GET",
			wantSegments: 2,
		},
		{
			name:         "post",
			rule:         &annotations.HttpRule{Pattern: &annotations.HttpRule_Post{Post: "/v1/x"}, Body: "*"},
			wantMethod:   "POST",
			wantSegments: 2,
			wantBody:     "*",
		},
		{
			name:         "put",
			rule:         &annotations.HttpRule{Pattern: &annotations.HttpRule_Put{Put: "/v1/x"}},
			wantMethod:   "PUT",
			wantSegments: 2,
		},
		{
			name:         "delete",
			rule:         &annotations.HttpRule{Pattern: &annotations.HttpRule_Delete{Delete: "/v1/x"}},
			wantMethod:   "DELETE",
			wantSegments: 2,
		},
		{
			name:         "patch",
			rule:         &annotations.HttpRule{Pattern: &annotations.HttpRule_Patch{Patch: "/v1/x"}},
			wantMethod:   "PATCH",
			wantSegments: 2,
		},
		{
			name: "custom_kind_with_verb",
			rule: &annotations.HttpRule{Pattern: &annotations.HttpRule_Custom{Custom: &annotations.CustomHttpPattern{
				Kind: "OPTIONS",
				Path: "/v1/x:custom",
			}}},
			wantMethod:   "OPTIONS",
			wantSegments: 2,
			wantVerb:     "custom",
		},
		{
			name:    "missing_pattern",
			rule:    &annotations.HttpRule{},
			wantErr: "http rule does not have an URL defined",
		},
		{
			name:    "empty_template",
			rule:    &annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: ""}},
			wantErr: "empty template string",
		},
		{
			name:    "malformed_template_duplicate_variable",
			rule:    &annotations.HttpRule{Pattern: &annotations.HttpRule_Get{Get: "/v1/{a}/{a}"}},
			wantErr: "variable 'a' bound multiple times",
		},
		{
			name: "additional_bindings",
			rule: &annotations.HttpRule{
				Pattern:           &annotations.HttpRule_Get{Get: "/v1/x"},
				AdditionalBindings: []*annotations.HttpRule{
					{Pattern: &annotations.HttpRule_Post{Post: "/v1/y"}},
				},
			},
			wantMethod:   "GET",
			wantSegments: 2,
		},
		{
			name: "additional_binding_missing_pattern",
			rule: &annotations.HttpRule{
				Pattern:           &annotations.HttpRule_Get{Get: "/v1/x"},
				AdditionalBindings: []*annotations.HttpRule{
					{Pattern: nil},
				},
			},
			wantErr: "parse additional binding 0",
		},
		{
			name: "additional_binding_malformed_template",
			rule: &annotations.HttpRule{
				Pattern:           &annotations.HttpRule_Get{Get: "/v1/x"},
				AdditionalBindings: []*annotations.HttpRule{
					{Pattern: &annotations.HttpRule_Get{Get: "/v1/{a}/{a}"}},
				},
			},
			wantErr: "parse additional binding 0",
		},
		{
			name:    "nil_rule",
			rule:    nil,
			wantErr: "http rule is nil",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRule(tt.rule)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantMethod, got.Method)
			assert.Equal(t, tt.wantSegments, len(got.Template.Segments))
			assert.Equal(t, tt.wantVerb, got.Template.Verb)
			assert.Equal(t, tt.wantBody, got.Body)
			if tt.name == "additional_bindings" {
				assert.Equal(t, 1, len(got.AdditionalRules))
				assert.Equal(t, "POST", got.AdditionalRules[0].Method)
				assert.Equal(t, "v1", got.AdditionalRules[0].Template.Segments[0].Literal)
			}
		})
	}
}
