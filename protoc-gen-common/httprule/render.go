package httprule

import "strings"

// String renders the template in its source-like documentation form, with
// variable segments shown as {field.path} placeholders and the verb, if any,
// appended. It is used for documentation output that advertises a binding's
// route shape without the variable match specifications.
func (t Template) String() string {
	parts := make([]string, 0, len(t.Segments))
	for _, seg := range t.Segments {
		switch seg.Kind {
		case SegmentKindVariable:
			parts = append(parts, "{"+seg.Variable.FieldPath.String()+"}")
		case SegmentKindLiteral:
			parts = append(parts, seg.Literal)
		case SegmentKindMatchSingle:
			parts = append(parts, "*")
		case SegmentKindMatchMultiple:
			parts = append(parts, "**")
		}
	}
	path := "/" + strings.Join(parts, "/")
	if t.Verb != "" {
		path += ":" + t.Verb
	}
	return path
}

// LiteralPath renders the template with every variable segment elided,
// wrapping the result with the caller's string quoting function. It is used
// for routes whose request fields are never path-bound, so templates passed
// here contain no variables in practice; the elision keeps the rendering total
// regardless.
func (t Template) LiteralPath(quote func(string) string) string {
	parts := make([]string, 0, len(t.Segments))
	for _, seg := range t.Segments {
		switch seg.Kind {
		case SegmentKindLiteral:
			parts = append(parts, seg.Literal)
		case SegmentKindMatchSingle:
			parts = append(parts, "*")
		case SegmentKindMatchMultiple:
			parts = append(parts, "**")
		}
	}
	path := "/" + strings.Join(parts, "/")
	if t.Verb != "" {
		path += ":" + t.Verb
	}
	return quote(path)
}

// HasVariables reports whether the template binds any field via a variable
// segment.
func (t Template) HasVariables() bool {
	for _, seg := range t.Segments {
		if seg.Kind == SegmentKindVariable {
			return true
		}
	}
	return false
}

// PathVariableFieldPaths returns the field path of every distinct field bound
// by a variable segment of the template, in segment order. Duplicate bindings
// are rejected at parse time, so every binding is distinct by construction.
func (t Template) PathVariableFieldPaths() []FieldPath {
	var paths []FieldPath
	seen := make(map[string]struct{})
	for _, seg := range t.Segments {
		if seg.Kind != SegmentKindVariable {
			continue
		}
		key := seg.Variable.FieldPath.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		paths = append(paths, seg.Variable.FieldPath)
	}
	return paths
}

// QueryExcluded reports whether the leaf field reached at path must be
// excluded from query parameter generation: the empty path, paths bound by a
// variable segment of this rule's template, and paths rooted at the rule's
// body field are all excluded.
func (r Rule) QueryExcluded(path FieldPath) bool {
	if len(path) == 0 {
		return true
	}
	covered := make(map[string]struct{})
	for _, segment := range r.Template.Segments {
		if segment.Kind == SegmentKindVariable {
			covered[segment.Variable.FieldPath.String()] = struct{}{}
		}
	}
	if _, ok := covered[path.String()]; ok {
		return true
	}
	return r.Body != "" && path[0] == r.Body
}
