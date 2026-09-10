package run

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/build"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

// CmdRun run project command.
var CmdRun = &cobra.Command{
	Use:   "run",
	Short: "Run service project",
	Long:  "Run service project. Example: gowind run admin",
	Run:   Run,
}

// Run service.
func Run(cmd *cobra.Command, args []string) {
	cmdArgs, _ := pkg.SplitArgs(cmd, args)

	inspector, err := pkg.NewModuleInspectorFromGo("")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return
	}

	// 先在模块根目录运行 `go mod tidy`
	if err = pkg.GoModTidy(cmd.Context(), inspector.Root); err != nil {
		return
	}

	var serviceName string

	if len(cmdArgs) > 0 {
		serviceName = strings.TrimSpace(cmdArgs[0])
		if serviceName == "" {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: service name is required\033[m\n")
			return
		}

		var valid bool
		valid, err = pkg.IsValidServiceName(inspector.Root, serviceName)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
			return
		}

		if !valid {
			err = fmt.Errorf("service '%s' does not exist or is not valid (missing cmd/server or configs)", serviceName)
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
			return
		}
	} else {
		// 未指定服务名称，检查当前目录是否为服务目录

		var wd string
		wd, err = os.Getwd()
		if err != nil {
			fmt.Printf("os.Getwd error: %v\n", err)
		}

		var hasCmd, hasConfigs bool
		hasCmd, hasConfigs, err = pkg.HasCmdAndConfigs(wd)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
			return
		}

		//log.Printf("[%s] hasCmd: %v, hasConfigs: %v\n", wd, hasCmd, hasConfigs)

		if hasCmd && hasConfigs {
			// 当前目录即为服务目录
			if err = runService(wd); err != nil {
				return
			}
			return
		}

		// 当前目录不是服务目录:枚举并一并运行模块内全部服务。
		names, listErr := pkg.ListServiceNames(inspector.Root)
		if listErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", listErr.Error())
			return
		}
		if len(names) == 0 {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: no valid services found under %s\033[m\n", filepath.Join(inspector.Root, "app"))
			return
		}
		if err = runAllServices(inspector.Root, names); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		}
		return
	}

	servicePath := path.Join(inspector.Root, "/app/", serviceName, "/service")

	if err = runService(servicePath); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
	}
}

// runService 运行服务，使用命令: go run ./cmd/server -c ./configs。
func runService(serviceWorkPath string) error {
	// 使用 pkg.NewGoCmd 执行 go run . [programArgs...]
	g := pkg.NewGoCmd(serviceWorkPath)
	g.Stdout = os.Stdout
	g.Stderr = os.Stderr

	// 构建并规范化路径
	appPath := filepath.Join(serviceWorkPath, "cmd", "server")
	appPathAbs, err := filepath.Abs(appPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}
	appPathAbs = filepath.Clean(appPathAbs)

	configPath := filepath.Join(serviceWorkPath, "configs")
	configPathAbs, err := filepath.Abs(configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}
	configPathAbs = filepath.Clean(configPathAbs)

	runArgs := []string{"run", appPathAbs, "-c", configPathAbs}

	if err = g.Run(runArgs...); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: %s\033[m\n", err.Error())
		return err
	}

	return nil
}

// runAllServices 一并运行模块内全部服务:先编译全部服务二进制,再并发拉起。
// 各服务输出按行加名称前缀;SIGINT/SIGTERM 统一停止全部进程。
// 直接执行二进制(而非 go run)使终止语义精确——杀掉的就是服务进程本身,
// 不会留下 go run 包装进程已死、服务进程仍存的孤儿。
func runAllServices(root string, names []string) error {
	type serviceProc struct {
		name    string
		svcDir  string
		binPath string
	}

	// 阶段一:编译全部服务(任一失败即中止,不启动任何进程)。
	procs := make([]serviceProc, 0, len(names))
	for _, name := range names {
		svcDir := filepath.Join(root, "app", name, "service")
		binPath, err := build.ResolveOutputPath(name, svcDir, runtime.GOOS, runtime.GOARCH, false)
		if err != nil {
			return fmt.Errorf("resolve output for service '%s' failed: %w", name, err)
		}
		if err = build.BuildBinary(svcDir, binPath, runtime.GOOS, runtime.GOARCH); err != nil {
			return fmt.Errorf("build for service '%s' failed: %w", name, err)
		}
		procs = append(procs, serviceProc{name: name, svcDir: svcDir, binPath: binPath})
	}

	// 信号上下文:中断/终止时由 exec 撤销全部子进程。
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 阶段二:并发拉起全部服务。
	// 终止语义:SIGINT/SIGTERM 撤销信号上下文,exec.CommandContext 随之杀掉各子进程;
	// 启动完成后另有显式 Kill 循环作冗余兜底,确保任何情形下子进程都被回收。
	stdoutMu := &sync.Mutex{}
	stderrMu := &sync.Mutex{}
	var wg sync.WaitGroup
	type startedProc struct {
		name string
		cmd  *exec.Cmd
	}
	startedProcs := make([]startedProc, 0, len(procs))
	for _, p := range procs {
		configPath := filepath.Join(p.svcDir, "configs")
		proc := exec.CommandContext(sigCtx, p.binPath, "-c", configPath)
		proc.Dir = p.svcDir
		proc.Stdout = newPrefixedWriter(os.Stdout, p.name, stdoutMu)
		proc.Stderr = newPrefixedWriter(os.Stderr, p.name, stderrMu)
		if err := proc.Start(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: failed to start service '%s': %s\033[m\n", p.name, err.Error())
			continue
		}
		startedProcs = append(startedProcs, startedProc{name: p.name, cmd: proc})
		wg.Add(1)
		go func(name string, proc *exec.Cmd) {
			defer wg.Done()
			err := proc.Wait()
			if sigCtx.Err() != nil {
				return // 整组停止,不逐个告警
			}
			_, _ = fmt.Fprintf(os.Stderr, "\033[33mWARNING: service '%s' exited: %v\033[m\n", name, err)
		}(p.name, proc)
	}

	if len(startedProcs) == 0 {
		return fmt.Errorf("no service could be started")
	}

	// 冗余兜底:信号上下文一旦取消,显式杀掉全部已启动子进程。
	go func() {
		<-sigCtx.Done()
		for _, sp := range startedProcs {
			if sp.cmd.Process != nil {
				_ = sp.cmd.Process.Kill()
			}
		}
	}()

	_, _ = fmt.Fprintf(os.Stdout, "\033[36mStarted %d service(s); Ctrl+C stops all of them.\033[m\n", len(startedProcs))

	wg.Wait()
	return nil
}

// prefixedWriter 给子进程输出逐行加服务名前缀。
// 同一底层流的多个实例共享互斥锁,保证行粒度不交叉。
type prefixedWriter struct {
	w      io.Writer
	prefix string
	mu     *sync.Mutex
	buf    []byte
}

func newPrefixedWriter(w io.Writer, name string, mu *sync.Mutex) *prefixedWriter {
	return &prefixedWriter{
		w:      w,
		prefix: "\033[36m[" + name + "]\033[0m ",
		mu:     mu,
	}
}

func (p *prefixedWriter) Write(b []byte) (int, error) {
	total := len(b)
	p.mu.Lock()
	defer p.mu.Unlock()
	for len(b) > 0 {
		nl := bytes.IndexByte(b, '\n')
		if nl < 0 {
			p.buf = append(p.buf, b...)
			break
		}
		p.buf = append(p.buf, b[:nl+1]...)
		_, _ = p.w.Write([]byte(p.prefix))
		_, _ = p.w.Write(p.buf)
		p.buf = p.buf[:0]
		b = b[nl+1:]
	}
	return total, nil
}
