package extract

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/tx7do/go-utils/stringcase"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

// FileCopy 一对复制动作。
type FileCopy struct {
	Src       string
	Dst       string
	Overwrite bool // 目标已存在,执行时将覆盖
}

// Plan 一次提取将发生的文件动作(只读计算,不写盘、不删除)。
type Plan struct {
	SourceService string
	TargetService string
	OrmType       string
	Models        []string
	KeepSource    bool

	// TargetWillBeCreated 目标服务不存在,执行时会先创建服务脚手架。
	TargetWillBeCreated bool

	// CopyFiles 源 -> 目标复制(含 import 改写)。
	CopyFiles []FileCopy

	// ModifiedFiles 目标/源端就地修改的文件(装配登记、server 注册等)。
	ModifiedFiles []string

	// DeletedFiles !KeepSource 时将被删除的源端文件。
	DeletedFiles []string

	// Warnings 计划期发现的问题(如源文件缺失,执行时将失败)。
	Warnings []string
}

// Plan 只读计算提取计划:枚举将复制、修改、删除的文件,不做任何变更。
// 供 --dry-run 预览与删除前确认。
func (e *Extractor) Plan() (*Plan, error) {
	p := &Plan{
		SourceService:       e.opts.SourceService,
		TargetService:       e.opts.TargetService,
		OrmType:             e.opts.OrmType,
		Models:              append([]string(nil), e.opts.Models...),
		KeepSource:          e.opts.KeepSource,
		TargetWillBeCreated: !isDirExists(e.targetServicePath()),
	}

	for _, model := range e.opts.Models {
		for _, pair := range e.modelFilePairs(model) {
			src, dst := pair[0], pair[1]
			if !pkg.IsFileExists(src) {
				p.Warnings = append(p.Warnings, fmt.Sprintf("missing source file (execution will fail): %s", src))
				continue
			}
			p.CopyFiles = append(p.CopyFiles, FileCopy{Src: src, Dst: dst, Overwrite: pkg.IsFileExists(dst)})
		}
	}

	// 目标端就地修改的候选文件(装配登记 / wire provider 集 / server 注册)。
	p.ModifiedFiles = append(p.ModifiedFiles, e.targetModifiedCandidates()...)

	if !e.opts.KeepSource {
		for _, model := range e.opts.Models {
			for _, pair := range e.modelFilePairs(model) {
				if pkg.IsFileExists(pair[0]) {
					p.DeletedFiles = append(p.DeletedFiles, pair[0])
				}
			}
		}
		p.ModifiedFiles = append(p.ModifiedFiles, e.sourceModifiedCandidates()...)
	}

	p.ModifiedFiles = dedupeSorted(p.ModifiedFiles)
	p.DeletedFiles = dedupeSorted(p.DeletedFiles)

	return p, nil
}

// modelFilePairs 返回一个模型的 (源文件, 目标文件) 复制对,
// 与 extractSchema/extractRepo/extractService 的命名规则一致。
func (e *Extractor) modelFilePairs(model string) [][2]string {
	snake := stringcase.SnakeCase(model)
	var pairs [][2]string

	switch e.opts.OrmType {
	case "ent":
		pairs = append(pairs, [2]string{
			filepath.Join(e.sourceServicePath(), "internal", "data", "ent", "schema", snake+".go"),
			filepath.Join(e.targetServicePath(), "internal", "data", "ent", "schema", snake+".go"),
		})
	case "gorm":
		pairs = append(pairs, [2]string{
			filepath.Join(e.sourceServicePath(), "internal", "data", "gorm", "schema", snake+".go"),
			filepath.Join(e.targetServicePath(), "internal", "data", "gorm", "schema", snake+".go"),
		})
		pairs = append(pairs, [2]string{
			filepath.Join(e.sourceServicePath(), "internal", "data", "gorm", "dao", snake+"_dao.go"),
			filepath.Join(e.targetServicePath(), "internal", "data", "gorm", "dao", snake+"_dao.go"),
		})
	}

	pairs = append(pairs, [2]string{
		filepath.Join(e.sourceServicePath(), "internal", "data", snake+"_repo.go"),
		filepath.Join(e.targetServicePath(), "internal", "data", snake+"_repo.go"),
	})
	pairs = append(pairs, [2]string{
		filepath.Join(e.sourceServicePath(), "internal", "service", snake+"_service.go"),
		filepath.Join(e.targetServicePath(), "internal", "service", snake+"_service.go"),
	})

	return pairs
}

// targetModifiedCandidates 目标端将被就地修改的现存文件。
// 装配形态与执行期一致:有 wiring 锚点用 wiring,否则 wire provider 集。
func (e *Extractor) targetModifiedCandidates() []string {
	var out []string
	target := e.targetServicePath()

	if wf := generators.FindWiringFile(filepath.Join(target, "cmd", "server"), ""); wf != "" {
		out = append(out, wf)
	} else if generators.WireProvidersExist(target) {
		for _, f := range []string{
			filepath.Join(target, "internal", "data", "providers", "wire_set.go"),
			filepath.Join(target, "internal", "service", "providers", "wire_set.go"),
		} {
			if pkg.IsFileExists(f) {
				out = append(out, f)
			}
		}
	}

	for _, f := range []string{
		filepath.Join(target, "internal", "server", "grpc_server.go"),
		filepath.Join(target, "internal", "server", "rest_server.go"),
	} {
		if pkg.IsFileExists(f) {
			out = append(out, f)
		}
	}
	return out
}

// sourceModifiedCandidates !KeepSource 时源端将被就地修改的现存文件
// (移除装配登记行 / wire provider)。
func (e *Extractor) sourceModifiedCandidates() []string {
	var out []string
	source := e.sourceServicePath()

	if wiringFiles, err := filepath.Glob(filepath.Join(source, "cmd", "server", "wiring*.go")); err == nil {
		out = append(out, wiringFiles...)
	}
	for _, f := range []string{
		filepath.Join(source, "internal", "data", "providers", "wire_set.go"),
		filepath.Join(source, "internal", "service", "providers", "wire_set.go"),
	} {
		if pkg.IsFileExists(f) {
			out = append(out, f)
		}
	}
	return out
}

func dedupeSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
