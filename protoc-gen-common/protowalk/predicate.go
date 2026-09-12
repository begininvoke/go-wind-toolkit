package protowalk

import "google.golang.org/protobuf/reflect/protoreflect"

// IsMessageCollectionField reports whether the field is a repeated or map
// field whose element or value type is a message. Such fields cannot be
// meaningfully serialized as query parameters.
func IsMessageCollectionField(field protoreflect.FieldDescriptor) bool {
	if field.IsList() && field.Kind() == protoreflect.MessageKind {
		return true
	}
	if field.IsMap() && field.MapValue().Kind() == protoreflect.MessageKind {
		return true
	}
	return false
}

// IsStreamingMethod reports whether the method streams on the client or the
// server side.
func IsStreamingMethod(method protoreflect.MethodDescriptor) bool {
	return method.IsStreamingClient() || method.IsStreamingServer()
}
