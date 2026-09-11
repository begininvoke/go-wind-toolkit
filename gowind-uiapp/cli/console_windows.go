//go:build windows

package cli

import (
	"os"
	"syscall"
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procFreeConsole   = kernel32.NewProc("FreeConsole")
	attachParent      = ^uintptr(0) // ATTACH_PARENT_PROCESS
)

// stdioValid 报告标准流句柄是否有效。GUI 子系统(-H windowsgui)构建的
// 程序在未重定向时句柄无效;一旦用户重定向到管道/文件,句柄有效。
func stdioValid(f *os.File) bool {
	if f == nil {
		return false
	}
	_, err := f.Stat()
	return err == nil
}

// attachParentConsole 让 GUI 子系统构建的可执行文件在命令行模式下
// 复用父进程的控制台,使 stdout/stderr 输出可见。
// 已附加控制台或输出被重定向时为无害空操作。
func attachParentConsole() {
	if stdioValid(os.Stdout) && stdioValid(os.Stderr) {
		return
	}
	_, _, _ = procFreeConsole.Call()
	if r, _, _ := procAttachConsole.Call(attachParent); r == 0 {
		return // 附加失败(如从资源管理器双击):保持静默
	}
	if !stdioValid(os.Stdout) {
		if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
			os.Stdout = f
		}
	}
	if !stdioValid(os.Stderr) {
		if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
			os.Stderr = f
		}
	}
}
