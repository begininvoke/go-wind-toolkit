package configexporter

import (
	"reflect"
	"testing"
)

func TestNormalizeEtcdEndpoints(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{in: "localhost:2379", want: []string{"localhost:2379"}},
		{in: "http://localhost:2379", want: []string{"localhost:2379"}},
		{in: "https://etcd.example.com:2379/", want: []string{"etcd.example.com:2379"}},
		{in: "http://n1:2379, n2:2379 ,http://n3:2379", want: []string{"n1:2379", "n2:2379", "n3:2379"}},
		{in: "", want: nil},
		{in: " , ,", want: nil},
	}
	for _, tt := range tests {
		got := normalizeEtcdEndpoints(tt.in)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("normalizeEtcdEndpoints(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
