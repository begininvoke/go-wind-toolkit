//go:build !windows

package cli

// attachParentConsole 在非 Windows 平台无需处理:标准 GUI 构建仍附带
// 终端标准流,CLI 模式输出天然可见。
func attachParentConsole() {}
