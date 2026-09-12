package main

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestGoldenExamples 用 buf 将 examples/proto 重新生成到临时目录,并与提交的
// golden 树(examples/proto/gen/dart)逐字节对比。生成器输出一旦漂移即失败,
// 防止"改了生成器忘了同步示例"或"示例被手改"。buf 不在 PATH 时跳过。
func TestGoldenExamples(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in -short mode")
	}
	if _, err := exec.LookPath("buf"); err != nil {
		t.Skip("buf not found in PATH; skipping golden regeneration check")
	}

	exDir := filepath.Join("examples", "proto")
	goldenDir := filepath.Join(exDir, "gen", "dart")

	bin := filepath.Join(t.TempDir(), "protoc-gen-dart-http")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := goBuild(bin); err != nil {
		t.Fatalf("build plugin: %v\n%s", err, out)
	}

	outDir := t.TempDir()
	tmpl := filepath.Join(t.TempDir(), "buf.gen.yaml")
	tmplBody := "version: v2\nplugins:\n  - local: " + bin + "\n    out: " + outDir + "\n"
	if err := os.WriteFile(tmpl, []byte(tmplBody), 0o600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cmd := exec.Command("buf", "generate", "--template", tmpl)
	cmd.Dir = exDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("buf generate: %v\n%s", err, out)
	}

	compareTrees(t, goldenDir, outDir)
}

// goBuild 构建本插件到 bin,返回输出(GOWORK=off 以隔离宿主杂散 go.work)。
func goBuild(bin string) ([]byte, error) {
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	return cmd.CombinedOutput()
}

// compareTrees 断言两目录树文件集合与内容完全一致。
func compareTrees(t *testing.T, goldenDir, outDir string) {
	t.Helper()

	readTree := func(root string) map[string][]byte {
		files := map[string][]byte{}
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			files[filepath.ToSlash(rel)] = data
			return nil
		})
		return files
	}

	golden := readTree(goldenDir)
	generated := readTree(outDir)

	missing := []string{}
	for rel := range golden {
		if _, ok := generated[rel]; !ok {
			missing = append(missing, rel)
		}
	}
	extra := []string{}
	for rel := range generated {
		if _, ok := golden[rel]; !ok {
			extra = append(extra, rel)
		}
	}
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("golden tree mismatch: missing=%v extra=%v", missing, extra)
	}

	diffs := []string{}
	for rel, want := range golden {
		if got := generated[rel]; !bytes.Equal(want, got) {
			diffs = append(diffs, rel)
		}
	}
	if len(diffs) > 0 {
		t.Fatalf("golden output drift in: %s\nRegenerate with buf in examples/proto and commit the result.",
			strings.Join(diffs, ", "))
	}
}
