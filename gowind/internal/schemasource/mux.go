// Package schemasource 为 SQL→schema 管线(sqlorm 的 ent 导入与 sqlproto
// 的 proto 转换)提供共享的数据源层:按 DSN scheme(mysql/postgres/text/file)
// 分发到对应 provider,返回带方言标记的 atlas Inspector 驱动;并承载两管线
// 逐字节同构的 DSN 归一化、SQL 文本载入与 atlas 类型解析工具。
package schemasource

import (
	"fmt"
	"io"
	"strings"

	"ariga.io/atlas/sql/schema"
)

type (
	// Mux 按 DSN scheme 路由到注册的 provider。
	Mux struct {
		providers map[string]func(string) (*Driver, error)
	}

	// Driver 包装 atlas Inspector 与其方言信息。
	Driver struct {
		io.Closer
		schema.Inspector
		Dialect    string
		SchemaName string
	}
)

// Close 关闭底层连接。text/file 等 DDL 文本驱动没有底层连接
// （内嵌 Closer 为 nil），直接返回成功避免 panic。
func (d *Driver) Close() error {
	if d.Closer != nil {
		return d.Closer.Close()
	}
	return nil
}

// New 返回新的 Mux。
func New() *Mux {
	return &Mux{
		providers: make(map[string]func(string) (*Driver, error)),
	}
}

// Default 是进程级共享的默认多路复用器,providers 由本包 init 注册。
var Default = New()

// RegisterProvider 为给定 scheme 注册 provider,供测试注入 mock 驱动。
func (u *Mux) RegisterProvider(p func(string) (*Driver, error), scheme ...string) {
	for _, s := range scheme {
		u.providers[s] = p
	}
}

// Open 打开 dsn 指向的数据源驱动。
func (u *Mux) Open(dsn string) (*Driver, error) {
	scheme, host, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %v", err)
	}
	p, ok := u.providers[scheme]
	if !ok {
		return nil, fmt.Errorf("provider does not exist: %q", scheme)
	}
	return p(host)
}

func parseDSN(url string) (string, string, error) {
	a := strings.SplitN(url, "://", 2)
	if len(a) != 2 {
		return "", "", fmt.Errorf(`failed to parse dsn: "%s"`, url)
	}
	return a[0], a[1], nil
}
