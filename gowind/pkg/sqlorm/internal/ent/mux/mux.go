package mux

import (
	"fmt"
	"io"
	"strings"

	"ariga.io/atlas/sql/schema"
)

type (
	// importProvider - returns an ImportDriver for a given dialect.
	importProvider func(string) (*ImportDriver, error)

	// Mux is used for routing dsn to correct provider.
	Mux struct {
		providers map[string]importProvider
	}

	// ImportDriver implements Inspector interface and holds inspection information.
	ImportDriver struct {
		io.Closer
		schema.Inspector
		Dialect    string
		SchemaName string
	}
)

// Close 关闭底层连接。text/file 等 DDL 文本驱动没有底层连接
// （内嵌 Closer 为 nil），直接返回成功避免 panic。
func (d *ImportDriver) Close() error {
	if d.Closer != nil {
		return d.Closer.Close()
	}
	return nil
}

// New returns a new Mux.
func New() *Mux {
	return &Mux{
		providers: make(map[string]importProvider),
	}
}

var Default = New()

// RegisterProvider is used to register an Atlas provider by key.
func (u *Mux) RegisterProvider(p importProvider, scheme ...string) {
	for _, s := range scheme {
		u.providers[s] = p
	}
}

// OpenImport is used for opening an import driver on a specific data source.
func (u *Mux) OpenImport(dsn string) (*ImportDriver, error) {
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
