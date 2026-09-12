package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// runProtoc 在测试 CWD(模块根目录)下按原始参数形状执行一次 protoc,
// 产物写入 genDir。要求 protoc 与所需插件已在 PATH 中。
func runProtoc(t *testing.T, genDir string, extraArgs ...string) {
	t.Helper()
	args := append([]string{"--experimental_allow_proto3_optional"}, extraArgs...)
	args = append(args, "-I", ".", filepath.Join("testdata", "integration", "test.proto"))
	out, err := exec.Command("protoc", args...).CombinedOutput()
	require.NoError(t, err, "protoc %v: %s", extraArgs, string(out))
}

// TestIntegrationGolden 把 testdata/integration/test.proto 的脱敏代码用
// protoc 生成到临时目录,并与提交的 golden(testdata/integration/
// test.pb.redact.go)逐字节对比:生成器输出一旦漂移即失败,防止"改了
// 生成器忘了同步示例"或"示例被手改"。全程不触碰仓库文件;protoc 或
// 依赖的 Go 插件不在 PATH 时跳过。
func TestIntegrationGolden(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	for _, tool := range []string{"protoc", "protoc-gen-go", "protoc-gen-go-grpc"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not found in PATH; skipping golden regeneration check", tool)
		}
	}

	// 把被测插件编译进临时目录。
	bin := filepath.Join(t.TempDir(), "protoc-gen-redact")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	buildCmd := exec.Command("go", "build", "-o", bin, ".")
	buildCmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build plugin: %v\n%s", err, string(out))
	}

	// 生成物(含 protoc-gen-go 产物)统一落在临时目录;source_relative
	// 会按 proto 源路径在输出根下镜像目录结构。
	genDir := t.TempDir()
	runProtoc(t, genDir,
		"--go_out="+genDir, "--go_opt=paths=source_relative",
		"--go-grpc_out="+genDir, "--go-grpc_opt=paths=source_relative",
	)
	runProtoc(t, genDir,
		"--plugin=protoc-gen-redact="+bin,
		"--redact_out="+genDir, "--redact_opt=paths=source_relative",
	)

	generated := filepath.Join(genDir, "testdata", "integration", "test.pb.redact.go")
	require.FileExists(t, generated, "plugin should have produced the redaction file")
	got, err := os.ReadFile(generated)
	require.NoError(t, err)

	goldenPath := filepath.Join("testdata", "integration", "test.pb.redact.go")
	want, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "committed golden is missing; regenerate it after intentional generator changes and commit it together")

	if !bytes.Equal(got, want) {
		i := 0
		for i < len(got) && i < len(want) && got[i] == want[i] {
			i++
		}
		t.Fatalf("generated redaction code drifted from the committed golden (first difference at byte %d; got %d bytes, want %d bytes). If the change is intentional, regenerate testdata/integration/test.pb.redact.go and commit both together", i, len(got), len(want))
	}
}

// TestIntegrationGoldenCompiles 提交的 golden 包应当始终可编译;golden
// 与生成物逐字节一致,故等价于校验生成物自身的可编译性。
func TestIntegrationGoldenCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	cmd := exec.Command("go", "build", "./testdata/integration")
	cmd.Env = append(os.Environ(), "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("committed golden package must compile: %v\n%s", err, string(out))
	}
}
