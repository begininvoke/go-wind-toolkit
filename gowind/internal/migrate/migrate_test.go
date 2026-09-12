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

func TestRenderVersionedProgram(t *testing.T) {
	// atlas 格式:不引入 sqltool。
	atlasProg, err := renderVersionedProgram(versionedProgramArgs{
		ImportPath:   "example.com/p/app/admin/service/internal/data/ent/migrate",
		DialectConst: "dialect.MySQL",
		DriverImport: `_ "github.com/go-sql-driver/mysql"`,
		DirCall:      "atlas.NewLocalDir",
		FormatCall:   "atlas.DefaultFormatter",
	})
	if err != nil {
		t.Fatalf("render atlas variant: %v", err)
	}
	for _, want := range []string{
		`migrate "example.com/p/app/admin/service/internal/data/ent/migrate"`,
		`atlas "ariga.io/atlas/sql/migrate"`,
		`atlas.NewLocalDir(os.Args[3])`,
		`schema.WithDialect(dialect.MySQL)`,
		`schema.WithFormatter(atlas.DefaultFormatter)`,
		`schema.WithMigrationMode(schema.ModeReplay)`,
		`migrate.NamedDiff(ctx, os.Args[1], os.Args[2],`,
	} {
		if !strings.Contains(atlasProg, want) {
			t.Fatalf("atlas variant missing %q:\n%s", want, atlasProg)
		}
	}
	if strings.Contains(atlasProg, "sqltool") {
		t.Fatalf("atlas variant must not import sqltool:\n%s", atlasProg)
	}

	// golang-migrate 格式:引入 sqltool 并使用其 Dir/Formatter。
	gmProg, err := renderVersionedProgram(versionedProgramArgs{
		ImportPath:   "example.com/p/app/admin/service/internal/data/ent/migrate",
		DialectConst: "dialect.PostgreSQL",
		DriverImport: `_ "github.com/lib/pq"`,
		DirCall:      "sqltool.NewGolangMigrateDir",
		FormatCall:   "sqltool.GolangMigrateFormatter",
		UsesSQLTool:  true,
	})
	if err != nil {
		t.Fatalf("render golang-migrate variant: %v", err)
	}
	for _, want := range []string{
		`"ariga.io/atlas/sql/sqltool"`,
		`sqltool.NewGolangMigrateDir(os.Args[3])`,
		`schema.WithFormatter(sqltool.GolangMigrateFormatter)`,
		`schema.WithDialect(dialect.PostgreSQL)`,
		`migrate.NamedDiff(ctx, os.Args[1], os.Args[2],`,
	} {
		if !strings.Contains(gmProg, want) {
			t.Fatalf("golang-migrate variant missing %q:\n%s", want, gmProg)
		}
	}
	// sqltool 变体不得引入未使用的 atlas/migrate 别名导入。
	if strings.Contains(gmProg, `atlas "ariga.io/atlas/sql/migrate"`) {
		t.Fatal("sqltool variant must not import atlas/migrate")
	}

	// 运行期输入必须全部经 os.Args 传入,模板不得内嵌占位之外的任何值。
	if strings.Contains(atlasProg, "mysql://") || strings.Contains(gmProg, "postgres://") {
		t.Fatal("versioned program must not embed connection URLs")
	}
}

func TestSanitizeMigName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "changes", want: "changes"},
		{in: "add-user-table", want: "add-user-table"},
		{in: "Add User Table!", want: "add-user-table"},
		{in: "  init_42  ", want: "init_42"},
		{in: "../../etc/passwd", want: "etc-passwd"},
		{in: "***", want: "changes"},
		{in: "", want: "changes"},
	}
	for _, tt := range tests {
		if got := sanitizeMigName(tt.in); got != tt.want {
			t.Fatalf("sanitizeMigName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestVersionedFlagValidation(t *testing.T) {
	// runVersionedAll 的参数校验:缺 URL、未知工具、与 --out 互斥。
	// 直接构造调用路径较重,这里校验映射表完整性与归一化入口。
	for _, name := range []string{"atlas", "golang-migrate", "goose", "db-mate", "flyway", "liquibase"} {
		if _, ok := migTools[name]; !ok {
			t.Fatalf("migTools missing documented tool %q", name)
		}
	}
	if _, ok := migTools["not-a-tool"]; ok {
		t.Fatal("migTools must not contain unknown tools")
	}
	for _, d := range []string{"mysql", "postgres", "sqlite3"} {
		if _, ok := entDialectConst[d]; !ok {
			t.Fatalf("entDialectConst missing dialect %q", d)
		}
	}
}
