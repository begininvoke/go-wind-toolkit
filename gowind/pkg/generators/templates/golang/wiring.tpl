package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
{{- if .ImportData}}
	"{{.Module}}/app/{{lower .Service}}/service/internal/data"
{{- end}}
{{- if .ImportClient}}
	"{{.Module}}/app/{{lower .Service}}/service/internal/data/client"
{{- end}}
{{- if .ImportServer}}
	"{{.Module}}/app/{{lower .Service}}/service/internal/server"
{{- end}}
{{- if .ImportServicePkg}}
	"{{.Module}}/app/{{lower .Service}}/service/internal/service"
{{- end}}
)

{{.HeaderComment}}
func initApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	// cleanup 注册表:rollback 时逆序执行。
	// Cleanup registry; rollback runs entries in reverse order.
	var cleanups []func()
	rollback := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

{{.InfraBlock}}

{{.RepoBlock}}

{{.ServiceBlock}}

{{.TransportBlock}}

	return newApp(
{{.NewAppArgs}}	), rollback, nil
}
