package codegen

import "testing"

func TestFileP(t *testing.T) {
	f := new(File)
	f.P("class ", "Foo", " {")
	f.P()
	f.P("}")

	want := "class Foo {\n\n}\n"
	if got := string(f.Content()); got != want {
		t.Fatalf("Content() = %q; want %q", got, want)
	}
}

func TestFilePEmpty(t *testing.T) {
	f := new(File)
	if got := string(f.Content()); got != "" {
		t.Fatalf("empty file Content() = %q; want empty", got)
	}
}
