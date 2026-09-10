package main

import (
	"github.com/tx7do/go-wind-toolkit/gowind/internal/build"
)

func init() {
	rootCmd.AddCommand(build.CmdBuild)
}
