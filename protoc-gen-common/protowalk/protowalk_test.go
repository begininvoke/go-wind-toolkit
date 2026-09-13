package protowalk_test

import (
	"fmt"
	"sort"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/protowalk"
)

// testFile 合成一个覆盖嵌套消息、嵌套枚举、map 字段、重复消息字段与
// 含流式方法服务的 FileDescriptor,供遍历/谓词测试使用。
func testFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()

	str := func(s string) *string { return proto.String(s) }
	i32 := func(n int32) *int32 { return proto.Int32(n) }
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
	opt := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	rep := descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	msg := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	str32 := descriptorpb.FieldDescriptorProto_TYPE_STRING

	fdp := &descriptorpb.FileDescriptorProto{
		Name:    str("pwtest.proto"),
		Package: str("pwtest"),
		Syntax:  str("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: str("Outer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("plain", 1, str32, opt, ""),
					field("inner", 2, msg, opt, ".pwtest.Outer.Inner"),
					field("tags", 3, str32, rep, ""),
					field("inners", 4, msg, rep, ".pwtest.Outer.Inner"),
					field("innerMap", 5, msg, rep, ".pwtest.Outer.InnerMapEntry"),
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{Name: str("Inner")},
					{
						Name: str("InnerMapEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							field("key", 1, str32, opt, ""),
							field("value", 2, msg, opt, ".pwtest.Outer.Inner"),
						},
						Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
					},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: str("Kind"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: str("KIND_A"), Number: i32(0)},
						},
					},
				},
			},
			{
				Name: str("Pet"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("outer", 1, msg, opt, ".pwtest.Outer"),
				},
			},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: str("Color"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: str("RED"), Number: i32(0)},
					{Name: str("GREEN"), Number: i32(1)},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: str("Svc"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: str("Unary"), InputType: str(".pwtest.Outer"), OutputType: str(".pwtest.Pet")},
					{Name: str("Stream"), InputType: str(".pwtest.Outer"), OutputType: str(".pwtest.Pet"), ClientStreaming: proto.Bool(true)},
					{Name: str("ServerStream"), InputType: str(".pwtest.Outer"), OutputType: str(".pwtest.Pet"), ServerStreaming: proto.Bool(true)},
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

func collectNames(t *testing.T, fd protoreflect.FileDescriptor) []string {
	t.Helper()
	var names []string
	protowalk.WalkFiles([]protoreflect.FileDescriptor{fd}, func(desc protoreflect.Descriptor) bool {
		names = append(names, string(desc.FullName()))
		return true
	})
	return names
}

func TestWalkFilesVisitsWholeTree(t *testing.T) {
	fd := testFile(t)

	got := collectNames(t, fd)

	wantSet := map[string]bool{}
	for _, n := range []string{
		"pwtest",
		// 遍历器止于枚举粒度,不访问枚举值(Color.RED 等)。
		"pwtest.Color",
		"pwtest.Outer",
		"pwtest.Outer.plain", "pwtest.Outer.inner", "pwtest.Outer.tags", "pwtest.Outer.inners", "pwtest.Outer.innerMap",
		"pwtest.Outer.Inner",
		"pwtest.Outer.InnerMapEntry", "pwtest.Outer.InnerMapEntry.key", "pwtest.Outer.InnerMapEntry.value",
		"pwtest.Outer.Kind",
		"pwtest.Pet", "pwtest.Pet.outer",
		"pwtest.Svc", "pwtest.Svc.Unary", "pwtest.Svc.Stream", "pwtest.Svc.ServerStream",
	} {
		wantSet[n] = true
	}

	gotSet := map[string]bool{}
	for _, n := range got {
		if gotSet[n] {
			t.Fatalf("descriptor %q visited more than once: %v", n, got)
		}
		gotSet[n] = true
	}
	if len(gotSet) != len(wantSet) {
		var missing, extra []string
		for n := range wantSet {
			if !gotSet[n] {
				missing = append(missing, n)
			}
		}
		for n := range gotSet {
			if !wantSet[n] {
				extra = append(extra, n)
			}
		}
		sort.Strings(missing)
		sort.Strings(extra)
		t.Fatalf("visited set mismatch; missing=%v extra=%v", missing, extra)
	}
}

func TestWalkFilesDeduplicatesRepeatedFiles(t *testing.T) {
	fd := testFile(t)

	var count int
	protowalk.WalkFiles([]protoreflect.FileDescriptor{fd, fd}, func(desc protoreflect.Descriptor) bool {
		count++
		return true
	})

	once := collectNames(t, fd)
	if count != len(once) {
		t.Fatalf("same file passed twice visited %d descriptors; want %d (each descriptor once)", count, len(once))
	}
}

func TestWalkFilesStopsDescendingOnFalse(t *testing.T) {
	fd := testFile(t)

	visited := map[string]bool{}
	protowalk.WalkFiles([]protoreflect.FileDescriptor{fd}, func(desc protoreflect.Descriptor) bool {
		visited[string(desc.FullName())] = true
		// 返回 false 阻止继续下钻 Outer:其字段、嵌套类型与嵌套枚举都不应访问。
		return string(desc.FullName()) != "pwtest.Outer"
	})

	for _, n := range []string{"pwtest", "pwtest.Color", "pwtest.Outer", "pwtest.Pet", "pwtest.Svc"} {
		if !visited[n] {
			t.Fatalf("descriptor %q should have been visited", n)
		}
	}
	for _, n := range []string{"pwtest.Outer.plain", "pwtest.Outer.Inner", "pwtest.Outer.Kind", "pwtest.Outer.innerMap"} {
		if visited[n] {
			t.Fatalf("descriptor %q must not be visited after WalkFunc returns false for Outer", n)
		}
	}
}

func TestIsMessageCollectionField(t *testing.T) {
	fd := testFile(t)
	fields := fd.Messages().Get(0).Fields()

	cases := map[string]struct {
		name string
		want bool
	}{
		"singular scalar":      {name: "plain", want: false},
		"singular message":     {name: "inner", want: false},
		"repeated scalar":      {name: "tags", want: false},
		"repeated message":     {name: "inners", want: true},
		"map to message value": {name: "innerMap", want: true},
	}
	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			f := fields.ByName(protoreflect.Name(tt.name))
			if f == nil {
				t.Fatalf("field %q not found", tt.name)
			}
			if got := protowalk.IsMessageCollectionField(f); got != tt.want {
				t.Fatalf("IsMessageCollectionField(%q) = %v; want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsStreamingMethod(t *testing.T) {
	fd := testFile(t)
	methods := fd.Services().Get(0).Methods()

	for _, tt := range []struct {
		name string
		want bool
	}{
		{name: "Unary", want: false},
		{name: "Stream", want: true},
		{name: "ServerStream", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := methods.ByName(protoreflect.Name(tt.name))
			if got := protowalk.IsStreamingMethod(m); got != tt.want {
				t.Fatalf("IsStreamingMethod(%s) = %v; want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestRangeHelpers(t *testing.T) {
	fd := testFile(t)

	var fieldNames []string
	protowalk.RangeFields(fd.Messages().Get(0), func(f protoreflect.FieldDescriptor) {
		fieldNames = append(fieldNames, string(f.Name()))
	})
	if fmt.Sprint(fieldNames) != fmt.Sprint([]string{"plain", "inner", "tags", "inners", "innerMap"}) {
		t.Fatalf("RangeFields order = %v", fieldNames)
	}

	var methodNames []string
	protowalk.RangeMethods(fd.Services().Get(0).Methods(), func(m protoreflect.MethodDescriptor) {
		methodNames = append(methodNames, string(m.Name()))
	})
	if fmt.Sprint(methodNames) != fmt.Sprint([]string{"Unary", "Stream", "ServerStream"}) {
		t.Fatalf("RangeMethods order = %v", methodNames)
	}

	var enumNames []string
	var lastFlags []bool
	protowalk.RangeEnumValues(fd.Enums().Get(0), func(v protoreflect.EnumValueDescriptor, last bool) {
		enumNames = append(enumNames, string(v.Name()))
		lastFlags = append(lastFlags, last)
	})
	if fmt.Sprint(enumNames) != fmt.Sprint([]string{"RED", "GREEN"}) {
		t.Fatalf("RangeEnumValues order = %v", enumNames)
	}
	if fmt.Sprint(lastFlags) != fmt.Sprint([]bool{false, true}) {
		t.Fatalf("RangeEnumValues last flags = %v", lastFlags)
	}
}
