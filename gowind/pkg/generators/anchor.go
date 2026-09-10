package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tx7do/go-utils/stringcase"
)

// register:* 锚点注释。注入式登记的落点,与参考工程(go-wind-admin / go-wind-cms)中
// tools/register 及其 wiring/server 文件内嵌的锚点字符串逐字节一致,勿改动。
const (
	AnchorRepo    = "// ── register:repo ── 新模块仓储在此行后注册(make register 工具锚点,勿删)"
	AnchorClient  = "// ── register:client ── 新模块服务客户端在此行后注册(make register 工具锚点,勿删)"
	AnchorService = "// ── register:service ── 新模块服务在此行后注册(make register 工具锚点,勿删)"
	AnchorRestArg = "// register:rest-arg ── 新模块服务实参在此行后追加(make register 工具锚点,勿删)"
	AnchorGrpcArg = "// register:grpc-arg ── 新模块服务实参在此行后追加(make register 工具锚点,勿删)"
	AnchorParam   = "// register:param ── 新模块服务形参在此行后注册(make register 工具锚点,勿删)"
	AnchorRoute   = "// register:route ── 新模块路由在此行后注册(make register 工具锚点,勿删)"
)

// WiringAnchorCandidates 注入 wiring 文件各登记位时可接受的锚点集合(按序探测,首个命中生效)。
// 不同服务形态的 wiring 文件带有不同的锚点:仓储型为 repo/grpc-arg,BFF 型为 client/rest-arg,
// 服务层锚点两形态同构。
func WiringAnchorCandidates(kind string) []string {
	switch kind {
	case "repo":
		return []string{AnchorRepo, AnchorClient}
	case "arg-rest":
		return []string{AnchorRestArg, AnchorGrpcArg}
	case "arg-grpc":
		return []string{AnchorGrpcArg, AnchorRestArg}
	default:
		return []string{AnchorService}
	}
}

// AnchorPatch 描述一次锚点注入:
// Path 目标文件;Anchors 候选锚点行(按去空白后的全文匹配,首个命中者生效);
// Lines 插入的行(不含行尾符);SkipIf 文件已包含该子串时跳过(幂等)。
type AnchorPatch struct {
	Path    string
	Anchors []string
	Lines   []string
	SkipIf  string
}

// ApplyAnchorPatches 依次应用各锚点注入。任一锚点缺失即报错(漏锚点应显式失败而非静默跳过)。
func ApplyAnchorPatches(patches ...AnchorPatch) error {
	for _, p := range patches {
		if err := applyAnchorPatch(p); err != nil {
			return err
		}
	}
	return nil
}

func applyAnchorPatch(p AnchorPatch) error {
	if p.Path == "" {
		return fmt.Errorf("anchor patch: empty target path")
	}
	if len(p.Anchors) == 0 {
		return fmt.Errorf("anchor patch %s: no anchors", p.Path)
	}

	raw, err := os.ReadFile(p.Path)
	if err != nil {
		return err
	}
	content := string(raw)
	if p.SkipIf != "" && strings.Contains(content, p.SkipIf) {
		return nil // 已登记,幂等跳过
	}

	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}
	lines := strings.Split(content, eol)

	for _, anchor := range p.Anchors {
		at := indexTrimmedLine(lines, anchor)
		if at < 0 {
			continue
		}
		out := make([]string, 0, len(lines)+len(p.Lines))
		out = append(out, lines[:at+1]...)
		out = append(out, p.Lines...)
		out = append(out, lines[at+1:]...)
		if err = os.WriteFile(p.Path, []byte(strings.Join(out, eol)), 0o644); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("%s: 未找到注册锚点 %q,请检查锚点注释是否被移动或删除", p.Path, p.Anchors[0])
}

func indexTrimmedLine(lines []string, anchor string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == anchor {
			return i
		}
	}
	return -1
}

// ContentHasTrimmedLine 报告内容中是否存在(去空白后)与 anchor 完全一致的行。
func ContentHasTrimmedLine(content string, anchor string) bool {
	return indexTrimmedLine(strings.Split(content, "\n"), anchor) >= 0 ||
		indexTrimmedLine(strings.Split(content, "\r\n"), anchor) >= 0
}

// FileHasTrimmedLine 报告文件中是否存在(去空白后)与 anchor 完全一致的行。
func FileHasTrimmedLine(path string, anchor string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return ContentHasTrimmedLine(string(raw), anchor)
}

// FindWiringFile 在 cmd/server 目录下定位手写装配文件(wiring*.go)。
// 仅把包含任一 wiring 侧登记锚点(repo/client/service)的文件视为装配文件;
// ormPreference 非空时优先返回其中构造了对应 ORM 客户端的文件
// (双文件形态如 wiring_ent.go/wiring_gorm.go 互斥构建标签),
// 否则回退到首个含锚点者;找不到返回空串。
func FindWiringFile(cmdServerDir string, ormPreference string) string {
	entries, err := filepath.Glob(filepath.Join(cmdServerDir, "wiring*.go"))
	if err != nil || len(entries) == 0 {
		return ""
	}

	var fallback string
	for _, f := range entries {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		content := string(raw)
		if !ContentHasTrimmedLine(content, AnchorRepo) &&
			!ContentHasTrimmedLine(content, AnchorClient) &&
			!ContentHasTrimmedLine(content, AnchorService) {
			continue
		}
		if ormPreference != "" &&
			strings.Contains(content, "New"+stringcase.UpperCamelCase(ormPreference)+"Client(") {
			return f
		}
		if fallback == "" {
			fallback = f
		}
	}
	return fallback
}

// DetectOrmClientVar 从既有装配文件中识别 ORM 客户端变量的实际命名
// (如 entClient / client 等),供注入的仓储构造行引用;识别失败回退规范命名。
func DetectOrmClientVar(wiringFile string, orm string) string {
	canonical := stringcase.LowerCamelCase(orm) + "Client"
	if wiringFile == "" {
		return canonical
	}
	raw, err := os.ReadFile(wiringFile)
	if err != nil {
		return canonical
	}
	pattern := regexp.MustCompile(
		`([A-Za-z_][A-Za-z0-9_]*)` +
			`(?:\s*,\s*[A-Za-z_][A-Za-z0-9_]*)*` +
			`\s*:=\s*[A-Za-z0-9_.]*New` +
			regexp.QuoteMeta(stringcase.UpperCamelCase(orm)) + `Client\s*\(`)
	if m := pattern.FindStringSubmatch(string(raw)); m != nil {
		return m[1]
	}
	return canonical
}

// RemoveWiringModuleLines 从装配文件中移除指定模块的全部登记行
// (仓储/客户端构造行、服务构造行、传输层实参行),供模块迁出时清理源端。
func RemoveWiringModuleLines(wiringFile string, model string) error {
	if wiringFile == "" {
		return nil
	}
	raw, err := os.ReadFile(wiringFile)
	if err != nil {
		return err
	}
	content := string(raw)

	camel := stringcase.LowerCamelCase(model)
	pascal := stringcase.UpperCamelCase(model)

	repoPrefix := camel + "Repo := data.New" + pascal + "Repo("
	servicePrefix := camel + "Service := service.New" + pascal + "Service("
	argExact := camel + "Service,"

	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}
	lines := strings.Split(content, eol)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, repoPrefix) ||
			strings.HasPrefix(trimmed, servicePrefix) ||
			trimmed == argExact {
			continue
		}
		out = append(out, line)
	}
	if len(out) == len(lines) {
		return nil
	}
	return os.WriteFile(wiringFile, []byte(strings.Join(out, eol)), 0o644)
}

// WireProvidersExist 报告服务目录下是否存在任一 wire provider 集
// (internal/{data,service,server}/providers/wire_set.go),用于识别旧式 wire 形态的服务。
func WireProvidersExist(serviceDir string) bool {
	for _, layer := range []string{"data", "service", "server"} {
		if _, err := os.Stat(filepath.Join(serviceDir, "internal", layer, "providers", "wire_set.go")); err == nil {
			return true
		}
	}
	return false
}
