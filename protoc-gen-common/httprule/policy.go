package httprule

import (
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/protowalk"
)

// HasQueryParams reports whether the method input, combined with the rule,
// has any field the generated client will render as a query parameter. Fields
// bound by path variables, fields of the body, and message-collection fields
// are not query parameters; a wildcard body leaves no fields for them.
func HasQueryParams(input protoreflect.MessageDescriptor, r Rule, isWellKnown func(protoreflect.Descriptor) bool) bool {
	if r.Body == "*" {
		return false
	}
	found := false
	WalkJSONLeafFields(input, isWellKnown, func(path FieldPath, field protoreflect.FieldDescriptor) {
		if found {
			return
		}
		if r.QueryExcluded(path) {
			return
		}
		if protowalk.IsMessageCollectionField(field) {
			return
		}
		found = true
	})
	return found
}

// MethodUsesRequest reports whether the method's generated client code will
// reference the request parameter at all, through a path variable, a body, or
// a query parameter.
func MethodUsesRequest(r Rule, input protoreflect.MessageDescriptor, isWellKnown func(protoreflect.Descriptor) bool) bool {
	return r.Template.HasVariables() || r.Body != "" || HasQueryParams(input, r, isWellKnown)
}

// SupportedMethod reports whether a generator following this package's
// binding model can handle the method, along with a human-readable reason
// when it cannot.
func SupportedMethod(method protoreflect.MethodDescriptor) (bool, string) {
	_, ok := Get(method)
	if !ok {
		return false, "no http rule annotation (google.api.http)"
	}
	if method.IsStreamingClient() && !method.IsStreamingServer() {
		return false, "client-only streaming is not supported"
	}
	return true, ""
}

// DefaultHost reads the google.api.default_host extension from a service
// descriptor.
func DefaultHost(service protoreflect.ServiceDescriptor) string {
	if service.Options() == nil {
		return ""
	}
	ext := proto.GetExtension(service.Options(), annotations.E_DefaultHost)
	if host, ok := ext.(string); ok && host != "" {
		return host
	}
	return ""
}

// FirstDefaultHost returns the first non-empty default_host across all
// services in files. Conflicting hosts are reported through warn; the first
// one wins.
func FirstDefaultHost(files []protoreflect.FileDescriptor, warn func(format string, args ...interface{})) string {
	var firstHost string
	var firstService string
	for _, file := range files {
		services := file.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			host := DefaultHost(svc)
			if host == "" {
				continue
			}
			if firstHost == "" {
				firstHost = host
				firstService = string(svc.FullName())
			} else if host != firstHost {
				warn("service %s has default_host %q but service %s already set %q; using the first one", svc.FullName(), host, firstService, firstHost)
			}
		}
	}
	return firstHost
}
