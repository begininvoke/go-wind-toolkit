package main

import (
	"github.com/tx7do/go-wind-toolkit/gowind/internal/migrate"
)

func init() {
	rootCmd.AddCommand(migrate.CmdMigrate)
}
