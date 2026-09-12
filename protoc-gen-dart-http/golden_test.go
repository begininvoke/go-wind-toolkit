package main

import (
	"testing"

	"github.com/tx7do/go-wind-toolkit/protoc-gen-common/goldentest"
)

// TestGoldenExamples 用 buf 将 examples/proto 重新生成到临时目录,并与提交的
// golden 树(examples/proto/gen/dart)逐字节对比。
func TestGoldenExamples(t *testing.T) {
	goldentest.Run(t, "protoc-gen-dart-http", "dart")
}
