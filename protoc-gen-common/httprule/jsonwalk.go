package httprule

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// JSONLeafWalkFunc receives every leaf field of the walked message, as the
// path of proto field names leading to it and the field descriptor itself.
type JSONLeafWalkFunc func(path FieldPath, field protoreflect.FieldDescriptor)

// WalkJSONLeafFields walks message and invokes f on every leaf field: fields
// of non-message kind, and message-kind fields whose type the embedding
// generator treats as a leaf, per isWellKnown. Nested non-leaf messages are
// walked recursively, with the field path extended by each traversal step.
func WalkJSONLeafFields(message protoreflect.MessageDescriptor, isWellKnown func(protoreflect.Descriptor) bool, f JSONLeafWalkFunc) {
	var w jsonWalker
	w.walkMessage(nil, message, isWellKnown, f)
}

type jsonWalker struct {
	seen map[protoreflect.FullName]struct{}
}

func (w *jsonWalker) enter(name protoreflect.FullName) bool {
	if _, ok := w.seen[name]; ok {
		return false
	}
	if w.seen == nil {
		w.seen = make(map[protoreflect.FullName]struct{})
	}
	w.seen[name] = struct{}{}
	return true
}

func (w *jsonWalker) walkMessage(path FieldPath, message protoreflect.MessageDescriptor, isWellKnown func(protoreflect.Descriptor) bool, f JSONLeafWalkFunc) {
	if w.enter(message.FullName()) {
		for i := 0; i < message.Fields().Len(); i++ {
			field := message.Fields().Get(i)
			p := append(FieldPath{}, path...)
			p = append(p, string(field.Name()))
			switch {
			case !field.IsMap() && !field.IsList() && field.Kind() == protoreflect.MessageKind:
				if field.Message() == nil {
					warnf("field %q has message kind but no valid message descriptor; treating as leaf", field.FullName())
					f(p, field)
					continue
				}
				if isWellKnown(field.Message()) {
					f(p, field)
				} else {
					w.walkMessage(p, field.Message(), isWellKnown, f)
				}
			default:
				f(p, field)
			}
		}
	}
}

func warnf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, "[httprule] WARN: "+format+"\n", args...)
}
