package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind-uiapp/internal/database"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "数据库连接与元数据",
}

func addDBFlags(cmd *cobra.Command) {
	cmd.Flags().String("dsn", "", "连接串（如 mysql://user:pass@tcp(localhost:3306)/db），可用环境变量 GOWIND_DSN；与离散参数二选一")
	cmd.Flags().String("type", "", "数据库类型: mysql | postgresql | sqlite | oracle（DSN 模式下可省略，按 scheme 推断）")
	cmd.Flags().String("host", "localhost", "主机地址（离散参数模式）")
	cmd.Flags().Int("port", 0, "端口（离散参数模式，按类型给默认值）")
	cmd.Flags().String("user", "", "用户名（离散参数模式）")
	cmd.Flags().String("password", "", "密码（离散参数模式）")
	cmd.Flags().String("database", "", "数据库名（离散参数模式）")
	cmd.Flags().String("db-path", "", "SQLite 文件路径（sqlite 模式）")
	cmd.Flags().Bool("ssl", false, "启用 SSL 连接")
}

// buildDBConfig 从 flag/环境变量构建数据库配置
func buildDBConfig(cmd *cobra.Command) database.DBConfig {
	dsn := flagString(cmd, "dsn", "")
	if dsn == "" {
		dsn = envOr("GOWIND_DSN", "")
	}
	if dsn != "" {
		dbType := flagString(cmd, "type", "")
		if dbType == "" {
			dbType = dbTypeFromDSN(dsn)
		}
		return database.DBConfig{
			Type:   database.DbType(dbType),
			UseDSN: true,
			DSN:    dsn,
		}
	}

	dbType := flagString(cmd, "type", "mysql")
	cfg := database.DBConfig{
		Type:     database.DbType(dbType),
		Host:     flagString(cmd, "host", "localhost"),
		Username: flagString(cmd, "user", ""),
		Password: flagString(cmd, "password", ""),
		Database: flagString(cmd, "database", ""),
		SSL:      boolFlag(cmd, "ssl"),
		DBPath:   flagString(cmd, "db-path", ""),
	}

	port, _ := cmd.Flags().GetInt("port")
	if port == 0 {
		port = defaultPort(dbType)
	}
	cfg.Port = port

	if cfg.Type == database.DbTypeOracle {
		checkErr(fmt.Errorf("oracle 连接暂不可用（上游驱动名错配），请使用 mysql/postgresql/sqlite 或提供 DDL 文件"))
	}
	return cfg
}

func dbTypeFromDSN(dsn string) string {
	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"), strings.HasPrefix(lower, "pgx://"):
		return string(database.DbTypePostgreSQL)
	case strings.HasPrefix(lower, "sqlite://"), strings.HasPrefix(lower, "sqlite:"):
		return string(database.DbTypeSQLite)
	case strings.HasPrefix(lower, "oracle://"):
		return string(database.DbTypeOracle)
	default:
		return string(database.DbTypeMySQL)
	}
}

func defaultPort(dbType string) int {
	switch dbType {
	case string(database.DbTypePostgreSQL):
		return 5432
	case string(database.DbTypeOracle):
		return 1521
	default:
		return 3306
	}
}

var dbTestCmd = &cobra.Command{
	Use:   "test",
	Short: "测试数据库连接",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := buildDBConfig(cmd)
		result, err := database.TestConnection(cfg)
		if err != nil {
			fail(err)
		}
		emit(result)
	},
}

var dbTablesCmd = &cobra.Command{
	Use:   "tables",
	Short: "列出数据库全部表",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := buildDBConfig(cmd)
		conn, err := database.Connect(cfg)
		if err != nil {
			fail(err)
		}
		defer conn.Close()

		tables, err := database.GetTables(conn, cfg.Type)
		if err != nil {
			fail(err)
		}
		emit(tables)
	},
}

var dbColumnsCmd = &cobra.Command{
	Use:   "columns",
	Short: "列出指定表的列信息",
	Run: func(cmd *cobra.Command, args []string) {
		table := flagString(cmd, "table", "")
		if table == "" {
			checkErr(fmt.Errorf("必须指定 --table"))
		}
		cfg := buildDBConfig(cmd)
		conn, err := database.Connect(cfg)
		if err != nil {
			fail(err)
		}
		defer conn.Close()

		columns, err := database.GetColumns(conn, cfg.Type, table)
		if err != nil {
			fail(err)
		}
		emit(columns)
	},
}

func init() {
	addDBFlags(dbTestCmd)
	addDBFlags(dbTablesCmd)
	addDBFlags(dbColumnsCmd)
	dbColumnsCmd.Flags().String("table", "", "表名（必填）")

	dbCmd.AddCommand(dbTestCmd, dbTablesCmd, dbColumnsCmd)
}
