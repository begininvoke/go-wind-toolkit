package schemasource

import (
	"os"
	"strings"

	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

// LoadSQLFromFile 从 schemaPath 载入 SQL 文本:若 schemaPath 指向普通文件
// 则读其内容,否则视作内联 DDL 文本,剥掉 text:// 等 scheme 前缀后原样返回。
func LoadSQLFromFile(schemaPath string) string {
	// 检查 schemaPath 是否为文件
	if !pkg.IsFileExists(schemaPath) {
		// 如果不是文件，去掉可能的 scheme 前缀（如 text://）后返回 SQL 文本内容。
		return stripScheme(schemaPath)
	}

	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return ""
	}

	return string(content)
}

// stripScheme 去掉 DSN 中的 scheme 前缀（如 text://, file://）
func stripScheme(path string) string {
	if idx := strings.Index(path, "://"); idx != -1 {
		return path[idx+3:]
	}
	return path
}

// ParseType 把 SQL 类型文本解析为 atlas schema 类型,依次尝试
// MySQL/PostgreSQL/SQLite 方言解析器,全部失败时返回 UnsupportedType。
func ParseType(raw string) (schema.Type, error) {
	mysqlType, err := mysql.ParseType(raw)
	if err == nil {
		return mysqlType, nil
	}

	postgresType, err := postgres.ParseType(raw)
	if err == nil {
		return postgresType, nil
	}

	sqliteType, err := sqlite.ParseType(raw)
	if err == nil {
		return sqliteType, nil
	}

	return &schema.UnsupportedType{T: raw}, nil
}
