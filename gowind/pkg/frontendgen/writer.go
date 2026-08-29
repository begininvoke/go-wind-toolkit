package frontendgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 写盘动作
const (
	ActionCreated    = "created"
	ActionOverwrite  = "overwritten"
	ActionMerged     = "merged"
	ActionMergedNew  = "merged-new" // 目标合并文件不存在，新建为独立片段文件
)

// WriteResult 单个文件的写盘结果
type WriteResult struct {
	// Path 写入的绝对/相对路径（outDir + 文件相对路径）
	Path string `json:"path"`
	// Action created / overwritten / merged / merged-new
	Action string `json:"action"`
	// Bytes 写入字节数
	Bytes int `json:"bytes"`
}

// WriteFiles 将生成文件写入目标目录。
//
// 普通文件创建或覆盖；vue-vben 的 page.*/menu.* 国际化片段是「合并式片段」：
// 目标 locales/langs/{lang}/page.json（或 menu.json）已存在时按键合并写回（merged），
// 不存在时新建独立片段文件（merged-new）。
func WriteFiles(files []GeneratedFile, outDir string) ([]WriteResult, error) {
	var results []WriteResult
	for _, file := range files {
		target := filepath.Join(outDir, filepath.FromSlash(file.Path))

		if isVbenLocaleFragment(file.Path, file.Content) {
			result, err := writeLocaleFragment(target, file.Content)
			if err != nil {
				return results, fmt.Errorf("写入 %s 失败: %w", target, err)
			}
			results = append(results, result)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return results, fmt.Errorf("创建目录 %s 失败: %w", filepath.Dir(target), err)
		}

		action := ActionCreated
		if _, err := os.Stat(target); err == nil {
			action = ActionOverwrite
		}

		if err := os.WriteFile(target, []byte(file.Content), 0o644); err != nil {
			return results, fmt.Errorf("写入 %s 失败: %w", target, err)
		}
		results = append(results, WriteResult{Path: target, Action: action, Bytes: len(file.Content)})
	}
	return results, nil
}

// isVbenLocaleFragment 判断是否为 vue-vben 的合并式国际化片段
// （路径形如 locales/langs/{lang}/page.{name}.json 或 menu.{name}.json，
// 且内容以两空格缩进的键开头，即无外层大括号的片段）
func isVbenLocaleFragment(path, content string) bool {
	normalized := filepath.ToSlash(path)
	if !strings.Contains(normalized, "locales/langs/") {
		return false
	}
	base := filepath.Base(normalized)
	if !strings.HasPrefix(base, "page.") && !strings.HasPrefix(base, "menu.") {
		return false
	}
	// page.json / menu.json 本体不是片段
	if base == "page.json" || base == "menu.json" {
		return false
	}
	return strings.HasPrefix(strings.TrimLeft(content, "\n"), "  \"")
}

// writeLocaleFragment 写入 vben 国际化片段：优先合并进同目录 page.json/menu.json
func writeLocaleFragment(fragmentPath, fragment string) (WriteResult, error) {
	dir := filepath.Dir(fragmentPath)
	base := filepath.Base(fragmentPath)             // page.role.json
	base = strings.TrimSuffix(base, ".json")        // page.role
	mergedBase := strings.SplitN(base, ".", 2)[0]   // page
	mergedPath := filepath.Join(dir, mergedBase+".json")

	content, err := os.ReadFile(mergedPath)
	if err != nil {
		// 目标合并文件不存在：新建独立片段文件（包一层大括号成为合法 JSON）
		if err := os.MkdirAll(filepath.Dir(fragmentPath), 0o755); err != nil {
			return WriteResult{}, err
		}
		wrapped := "{\n" + strings.TrimRight(fragment, "\n") + "\n}\n"
		if err := os.WriteFile(fragmentPath, []byte(wrapped), 0o644); err != nil {
			return WriteResult{}, err
		}
		return WriteResult{Path: fragmentPath, Action: ActionMergedNew, Bytes: len(wrapped)}, nil
	}

	merged, err := mergeJSONFragment(string(content), fragment)
	if err != nil {
		return WriteResult{}, err
	}
	if err := os.WriteFile(mergedPath, []byte(merged), 0o644); err != nil {
		return WriteResult{}, err
	}
	return WriteResult{Path: mergedPath, Action: ActionMerged, Bytes: len(merged)}, nil
}

// mergeJSONFragment 将无外层大括号的片段（`  "key": {...}`）按顶层键合并进 JSON 文本。
// 键已存在则原位替换，不存在则追加到对象末尾。基于行级处理保留原文件其余格式。
func mergeJSONFragment(target, fragment string) (string, error) {
	fragLines := strings.Split(strings.TrimRight(fragment, "\n"), "\n")

	// 片段顶层键（首行形如 `  "key": {`）
	topKey := ""
	for _, line := range fragLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "\"") {
			topKey = strings.SplitN(strings.TrimPrefix(trimmed, "\""), "\"", 2)[0]
			break
		}
	}
	if topKey == "" {
		return "", fmt.Errorf("片段中没有可识别的顶层键")
	}

	lines := strings.Split(strings.TrimRight(target, "\n"), "\n")

	// 找到末尾闭合大括号行
	lastBrace := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "}" {
			lastBrace = i
			break
		}
	}
	if lastBrace < 0 {
		return "", fmt.Errorf("目标文件不是 JSON 对象")
	}

	// 查找既有键的行区间（`  "key":` 开始，到同缩进级别的闭合行）
	start, end := -1, -1
	keyLine := "  \"" + topKey + "\":"
	for i := 0; i < lastBrace; i++ {
		if strings.HasPrefix(lines[i], keyLine) {
			start = i
			// 向后找回到 2 空格缩进的 `}` 或 `},` 行
			for j := i + 1; j < lastBrace; j++ {
				t := strings.TrimRight(lines[j], ",")
				if t == "  }" {
					end = j
					break
				}
			}
			break
		}
	}

	var out []string
	if start >= 0 && end >= start {
		// 原位替换
		out = append(out, lines[:start]...)
		out = append(out, fragLines...)
		if end < lastBrace-1 {
			out = append(out, lines[end+1:]...)
		} else {
			out = append(out, "}")
		}
	} else {
		// 追加到末尾（在最后一个条目后补逗号）
		out = append(out, lines[:lastBrace]...)
		if len(out) > 0 {
			last := strings.TrimRight(out[len(out)-1], " \t")
			if last != "{" && !strings.HasSuffix(last, ",") && !strings.HasSuffix(last, "{") {
				out[len(out)-1] = last + ","
			}
		}
		out = append(out, fragLines...)
		out = append(out, "}")
	}

	return strings.Join(out, "\n") + "\n", nil
}
