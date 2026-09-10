package sqlkratos

import (
	"path/filepath"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/generators"
)

// wiringContext 记录目标服务的依赖装配形态与注入落点。
//
// 形态判定(在生成开始时一次性完成,各阶段共用):
//   - 既有手写装配文件(含登记锚点) → 注入式登记(wiring 模式);
//   - 既无装配文件又无 wire provider 集(全新服务) → 渲染手写装配骨架(wiring 模式);
//   - 仅有 wire provider 集(旧式服务) → 沿用 wire 生成,向下兼容。
type wiringContext struct {
	useWiringDI bool
	// wiringFile 既有手写装配文件路径(注入落点);空表示本次将全新渲染装配骨架。
	wiringFile string
	// ormClientVar 既有装配文件中 ORM 客户端变量的实际命名;注入的仓储构造行引用它。
	ormClientVar string
	// isBff 为真表示 BFF 形态(纯 rest 且不落仓储),数据层为服务客户端而非仓储。
	isBff bool
}

func newWiringContext(outputPath string, serviceName string, orm string, useGrpc bool, useRepo bool) *wiringContext {
	serviceDir := filepath.Join(outputPath, "app", serviceName, "service")
	cmdServerDir := filepath.Join(serviceDir, "cmd", "server")

	isBff := !useGrpc && !useRepo

	// BFF 装配文件不含 ORM 客户端构造,按锚点直接探测;仓储型服务优先匹配
	// 构造了对应 ORM 客户端的装配文件(兼容 wiring_ent.go / wiring_gorm.go 双文件形态)。
	pref := ""
	if !isBff {
		pref = orm
	}

	wiringFile := generators.FindWiringFile(cmdServerDir, pref)
	wireMode := wiringFile == "" && generators.WireProvidersExist(serviceDir)

	ctx := &wiringContext{
		useWiringDI: !wireMode,
		wiringFile:  wiringFile,
		isBff:       isBff,
	}
	if wiringFile != "" && !isBff && orm != "" {
		ctx.ormClientVar = generators.DetectOrmClientVar(wiringFile, orm)
	}
	return ctx
}
