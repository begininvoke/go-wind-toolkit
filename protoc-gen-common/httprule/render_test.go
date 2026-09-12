package httprule

import (
	"testing"

	"gotest.tools/v3/assert"
)

func Test_TemplateString(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		input string
		want  string
	}{
		{input: "/v1/messages", want: "/v1/messages"},
		{input: "/v1/messages:peek", want: "/v1/messages:peek"},
		// The documentation form drops variable match specifications.
		{input: "/v1/{name=messages/*}:publish", want: "/v1/{name}:publish"},
		{input: "/{id=**}", want: "/{id}"},
		{input: "/v1/{name}/messages/{msg.name}", want: "/v1/{name}/messages/{msg.name}"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			tmpl, err := ParseTemplate(tt.input)
			assert.NilError(t, err)
			assert.Equal(t, tt.want, tmpl.String())
		})
	}
}

func Test_TemplateLiteralPath(t *testing.T) {
	t.Parallel()
	identity := func(s string) string { return s }
	for _, tt := range []struct {
		input string
		want  string
	}{
		{input: "/v1/messages", want: "/v1/messages"},
		{input: "/v1/messages:peek", want: "/v1/messages:peek"},
		// Variable segments are elided entirely, separators included.
		{input: "/{id=**}", want: "/"},
		{input: "/v1/{name=messages/*}/x", want: "/v1/x"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			tmpl, err := ParseTemplate(tt.input)
			assert.NilError(t, err)
			assert.Equal(t, tt.want, tmpl.LiteralPath(identity))
		})
	}
}

func Test_TemplateHasVariables(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		input string
		want  bool
	}{
		{input: "/v1/messages", want: false},
		{input: "/v1/{name}", want: true},
		{input: "/v1/{name=messages/*}", want: true},
		{input: "/{id=**}", want: true},
	} {
		t.Run(tt.input, func(t *testing.T) {
			tmpl, err := ParseTemplate(tt.input)
			assert.NilError(t, err)
			assert.Equal(t, tt.want, tmpl.HasVariables())
		})
	}
}

func Test_TemplatePathVariableFieldPaths(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		input string
		want  []FieldPath
	}{
		{
			input: "/test/{message.id}/{message.name=messages/*}",
			want:  []FieldPath{{"message", "id"}, {"message", "name"}},
		},
		{
			input: "/{id=**}",
			want:  []FieldPath{{"id"}},
		},
		{
			input: "/test/noparams",
			want:  nil,
		},
	} {
		t.Run(tt.input, func(t *testing.T) {
			tmpl, err := ParseTemplate(tt.input)
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.want, tmpl.PathVariableFieldPaths())
		})
	}
}

func Test_RuleQueryExcluded(t *testing.T) {
	t.Parallel()
	tmpl, err := ParseTemplate("/v1/{name=messages/*}")
	assert.NilError(t, err)

	ruleWithBody := Rule{Template: tmpl, Body: "foo"}
	ruleWithoutBody := Rule{Template: tmpl, Body: ""}

	for _, tt := range []struct {
		name string
		rule Rule
		path FieldPath
		want bool
	}{
		{name: "empty path", rule: ruleWithoutBody, path: FieldPath{}, want: true},
		{name: "path bound by variable", rule: ruleWithoutBody, path: FieldPath{"name"}, want: true},
		// Coverage matches the full dotted path, not its root.
		{name: "dotted path under bound root", rule: ruleWithoutBody, path: FieldPath{"name", "sub"}, want: false},
		{name: "unbound path without body", rule: ruleWithoutBody, path: FieldPath{"other"}, want: false},
		{name: "path rooted at body field", rule: ruleWithBody, path: FieldPath{"foo", "bar"}, want: true},
		{name: "unbound path with body", rule: ruleWithBody, path: FieldPath{"other"}, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.rule.QueryExcluded(tt.path))
		})
	}
}
