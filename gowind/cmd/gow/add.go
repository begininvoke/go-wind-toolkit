package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/service"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "add microservice to an existing project",
	Long:  "Add microservice to the current project. Example: gow add <name>  (shorthand for gow add service <name>)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments: expected 1 service name, got %d", len(args))
		}
		return service.Run(cmd, args)
	},
	SilenceUsage: true,
}

func init() {
	addCmd.AddCommand(service.CmdService)
	rootCmd.AddCommand(addCmd)
}
