package codegen

import (
	"sort"
	"testing"
)

func TestIndent(t *testing.T) {
	for _, tt := range []struct {
		n    int
		want string
	}{
		{n: 0, want: ""},
		{n: 1, want: "  "},
		{n: 3, want: "      "},
	} {
		if got := Indent(tt.n); got != tt.want {
			t.Fatalf("Indent(%d) = %q (len %d); want %q (len %d)", tt.n, got, len(got), tt.want, len(tt.want))
		}
	}
}

func TestLowerFirst(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "a", want: "a"},
		{in: "Abc", want: "abc"},
		{in: "ABC", want: "aBC"},
		{in: "already", want: "already"},
		{in: "1num", want: "1num"},
	} {
		if got := LowerFirst(tt.in); got != tt.want {
			t.Fatalf("LowerFirst(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestLocaleCompare(t *testing.T) {
	// 排序键仿真 JS localeCompare(UCA):标点 < 数字 < 字母,字母不区分大小写。
	for _, tt := range []struct {
		a, b string
		want bool // a 排在 b 之前
	}{
		{a: "a", b: "b", want: true},
		{a: "b", b: "a", want: false},
		{a: "a", b: "a", want: false}, // 严格小于,相等为 false
		{a: "apple", b: "Zebra", want: true},
		{a: "Zebra", b: "apple", want: false},
		{a: "_id", b: "id", want: true},          // 下划线先于字母
		{a: "userId", b: "user_id", want: false}, // 下划线先于字母:user_id 在前
		{a: "v1", b: "va", want: true},           // 数字先于字母
		{a: "a1", b: "aa", want: true},           // 位内数字先于字母
	} {
		if got := LocaleCompare(tt.a, tt.b); got != tt.want {
			t.Errorf("LocaleCompare(%q, %q) = %v; want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSortKeyOrdering(t *testing.T) {
	// 一组混合标识符按 sortKey 升序应得到标点 < 数字 < 字母、大小写不敏感
	// 的确定顺序——这是两侧生成器字段/枚举排序稳定性的依据。
	got := []string{"User", "_meta", "id2", "ID1", "zz"}
	sort.SliceStable(got, func(i, j int) bool { return LocaleCompare(got[i], got[j]) })
	want := []string{"_meta", "ID1", "id2", "User", "zz"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted = %v; want %v", got, want)
		}
	}
}
