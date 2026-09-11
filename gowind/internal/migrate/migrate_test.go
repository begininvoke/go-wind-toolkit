package migrate

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeDialect(t *testing.T) {
	tests := []struct {
		in   string
		want string
		err  bool
	}{
		{in: "", want: "mysql"},
		{in: "mysql", want: "mysql"},
		{in: "MySQL", want: "mysql"},
		{in: "postgres", want: "postgres"},
		{in: "postgresql", want: "postgres"},
		{in: "pg", want: "postgres"},
		{in: "sqlite", want: "sqlite3"},
		{in: "sqlite3", want: "sqlite3"},
		{in: "oracle", err: true},
		{in: "sqlserver", err: true},
	}
	for _, tt := range tests {
		got, err := normalizeDialect(tt.in)
		if tt.err {
			if err == nil {
				t.Fatalf("normalizeDialect(%q) expected error, got %q", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeDialect(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("normalizeDialect(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRenderDumpProgram(t *testing.T) {
	program, err := renderDumpProgram("example.com/myproject/app/admin/service/internal/data/ent/migrate")
	if err != nil {
		t.Fatalf("renderDumpProgram: %v", err)
	}
	for _, want := range []string{
		"package main",
		`migrate "example.com/myproject/app/admin/service/internal/data/ent/migrate"`,
		"schemadsl.DDL(",
		"Tables:  migrate.Tables,",
		"Dialect: dialect",
		"Version: version",
	} {
		if !strings.Contains(program, want) {
			t.Fatalf("dump program missing %q:\n%s", want, program)
		}
	}
	// 方言与版本必须走命令行参数,不得内嵌进源码。
	if strings.Contains(program, "mysql") {
		t.Fatalf("dump program must not embed dialect literals:\n%s", program)
	}
}

func TestOutputSQLPath(t *testing.T) {
	root := t.TempDir()

	// 默认:各服务 migrations/ 下,schema.<dialect>.sql。
	def, err := outputSQLPath(root, "admin", "postgres")
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	if want := filepath.Join(root, "app", "admin", "service", "migrations", "schema.postgres.sql"); def != want {
		t.Fatalf("default = %q, want %q", def, want)
	}

	// --out:集中目录,文件名带服务名避免覆盖。
	outDirFlag = t.TempDir()
	defer func() { outDirFlag = "" }()
	shared, err := outputSQLPath(root, "admin", "mysql")
	if err != nil {
		t.Fatalf("out dir: %v", err)
	}
	if want := filepath.Join(outDirFlag, "admin.mysql.sql"); shared != want {
		t.Fatalf("out dir = %q, want %q", shared, want)
	}
}
