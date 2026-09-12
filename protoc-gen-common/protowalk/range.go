package protowalk

import "google.golang.org/protobuf/reflect/protoreflect"

// RangeFields invokes f for every field of message, in declaration order.
func RangeFields(message protoreflect.MessageDescriptor, f func(field protoreflect.FieldDescriptor)) {
	for i := 0; i < message.Fields().Len(); i++ {
		f(message.Fields().Get(i))
	}
}

// RangeMethods invokes f for every method of methods, in declaration order.
func RangeMethods(methods protoreflect.MethodDescriptors, f func(method protoreflect.MethodDescriptor)) {
	for i := 0; i < methods.Len(); i++ {
		f(methods.Get(i))
	}
}

// RangeEnumValues invokes f for every value of enum, in declaration order.
// last reports whether the value is the enum's final one, which generated
// output uses for separator placement.
func RangeEnumValues(enum protoreflect.EnumDescriptor, f func(value protoreflect.EnumValueDescriptor, last bool)) {
	for i := 0; i < enum.Values().Len(); i++ {
		if i == enum.Values().Len()-1 {
			f(enum.Values().Get(i), true)
		} else {
			f(enum.Values().Get(i), false)
		}
	}
}
