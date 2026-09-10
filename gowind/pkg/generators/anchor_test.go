package generators

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, name string, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readTestFileRaw(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// 模板文件的工作区行尾随平台检出而变(LF/CRLF),断言按语义比较,统一归一为 LF。
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

func TestApplyAnchorPatchesInjectsAfterAnchor(t *testing.T) {
	f := writeTestFile(t, "wiring.go", "package main\n\nfunc f() {\n\t"+AnchorService+"\n\tother()\n}\n")

	err := ApplyAnchorPatches(AnchorPatch{
		Path:    f,
		Anchors: []string{AnchorService},
		Lines:   []string{"\tinjected := true"},
		SkipIf:  "injected := true",
	})
	if err != nil {
		t.Fatalf("ApplyAnchorPatches: %v", err)
	}

	content := readTestFile(t, f)
	want := AnchorService + "\n\tinjected := true\n"
	if !strings.Contains(content, want) {
		t.Fatalf("injected line not directly after anchor:\n%s", content)
	}
	if !strings.Contains(content, "\tother()\n") {
		t.Fatalf("content after insertion point lost:\n%s", content)
	}
}

func TestApplyAnchorPatchesIdempotent(t *testing.T) {
	f := writeTestFile(t, "wiring.go", "package main\n\nfunc f() {\n\t"+AnchorService+"\n}\n")

	patch := AnchorPatch{
		Path:    f,
		Anchors: []string{AnchorService},
		Lines:   []string{"\tservice.NewThingService("},
		SkipIf:  "service.NewThingService(",
	}
	if err := ApplyAnchorPatches(patch); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	afterFirst := readTestFile(t, f)
	if !strings.Contains(afterFirst, "service.NewThingService(") {
		t.Fatal("first apply did not inject")
	}

	if err := ApplyAnchorPatches(patch); err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if got := readTestFile(t, f); got != afterFirst {
		t.Fatal("second apply changed the file despite skipIf")
	}
}

func TestApplyAnchorPatchesCRLF(t *testing.T) {
	f := writeTestFile(t, "wiring.go", "package main\r\n\r\nfunc f() {\r\n\t"+AnchorService+"\r\n}\r\n")

	err := ApplyAnchorPatches(AnchorPatch{
		Path:    f,
		Anchors: []string{AnchorService},
		Lines:   []string{"\tinjected := true"},
		SkipIf:  "injected := true",
	})
	if err != nil {
		t.Fatalf("ApplyAnchorPatches: %v", err)
	}

	content := readTestFileRaw(t, f)
	if !strings.Contains(content, AnchorService+"\r\n\tinjected := true\r\n") {
		t.Fatalf("CRLF line endings not preserved:\n%q", content)
	}
}

func TestApplyAnchorPatchesMissingAnchor(t *testing.T) {
	f := writeTestFile(t, "wiring.go", "package main\n\nfunc f() {\n}\n")

	err := ApplyAnchorPatches(AnchorPatch{
		Path:    f,
		Anchors: []string{AnchorService},
		Lines:   []string{"\tx"},
		SkipIf:  "never-there",
	})
	if err == nil {
		t.Fatal("expected error for missing anchor")
	}
	if !strings.Contains(err.Error(), "未找到注册锚点") {
		t.Fatalf("error should name the missing anchor, got: %v", err)
	}
}

func TestApplyAnchorPatchesMultiAnchorCandidates(t *testing.T) {
	// BFF 形态文件只带 client 锚点;repo 注入应回退命中该锚点。
	f := writeTestFile(t, "wiring.go", "package main\n\nfunc f() {\n\t"+AnchorClient+"\n}\n")

	err := ApplyAnchorPatches(AnchorPatch{
		Path:    f,
		Anchors: []string{AnchorRepo, AnchorClient},
		Lines:   []string{"\tinjected := true"},
		SkipIf:  "injected := true",
	})
	if err != nil {
		t.Fatalf("ApplyAnchorPatches: %v", err)
	}
	if !strings.Contains(readTestFile(t, f), AnchorClient+"\n\tinjected := true\n") {
		t.Fatal("injection did not hit the client anchor")
	}
}

func TestFindWiringFilePrefersOrmConstructingFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "wiring_other.go")
	b := filepath.Join(dir, "wiring_gorm.go")
	if err := os.WriteFile(a, []byte("package main\n\t"+AnchorRepo+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("package main\n\t"+AnchorRepo+"\n gormClient, err := client.NewGormClient(ctx)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := FindWiringFile(dir, "gorm"); got != b {
		t.Fatalf("expected gorm-constructing file, got %q", got)
	}
	if got := FindWiringFile(dir, ""); got == "" {
		t.Fatal("expected fallback to first anchored wiring file for empty preference")
	}
}

func TestFindWiringFileIgnoresUnanchoredFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wiring.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := FindWiringFile(dir, ""); got != "" {
		t.Fatalf("expected no wiring file (no anchors), got %q", got)
	}
}

func TestDetectOrmClientVar(t *testing.T) {
	f := writeTestFile(t, "wiring.go",
		"package main\n\n\tclientVar, cleanupVar, err := data.NewEntClient(ctx)\n")
	if got := DetectOrmClientVar(f, "ent"); got != "clientVar" {
		t.Fatalf("expected clientVar, got %q", got)
	}
	if got := DetectOrmClientVar("", "gorm"); got != "gormClient" {
		t.Fatalf("expected canonical fallback, got %q", got)
	}
}

func TestRemoveWiringModuleLines(t *testing.T) {
	f := writeTestFile(t, "wiring.go", "package main\n\nfunc f() {\n"+
		"\tthingRepo := data.NewThingRepo(ctx, entClient)\n"+
		"\tkeepMe := 1\n"+
		"\tthingService := service.NewThingService(ctx, thingRepo)\n"+
		"\t\tthingService,\n"+
		"\t\totherService,\n"+
		"\t}\n")

	if err := RemoveWiringModuleLines(f, "thing"); err != nil {
		t.Fatalf("RemoveWiringModuleLines: %v", err)
	}
	content := readTestFile(t, f)
	for _, banned := range []string{"data.NewThingRepo(", "service.NewThingService(", "\t\tthingService,"} {
		if strings.Contains(content, banned) {
			t.Fatalf("line %q not removed:\n%s", banned, content)
		}
	}
	if !strings.Contains(content, "\tkeepMe := 1\n") || !strings.Contains(content, "\t\totherService,\n") {
		t.Fatalf("unrelated lines lost:\n%s", content)
	}
}

func TestWireProvidersExist(t *testing.T) {
	dir := t.TempDir()
	if WireProvidersExist(dir) {
		t.Fatal("empty dir should have no providers")
	}
	providers := filepath.Join(dir, "internal", "data", "providers")
	if err := os.MkdirAll(providers, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providers, "wire_set.go"), []byte("package providers\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !WireProvidersExist(dir) {
		t.Fatal("providers dir should be detected")
	}
}
