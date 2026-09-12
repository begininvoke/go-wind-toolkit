package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/project"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "create new project",
	Long:  "Create new project. Example: gow new <name>  (shorthand for gow new project <name>)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		if len(args) > 1 {
			return fmt.Errorf("too many arguments: expected 1 project name, got %d", len(args))
		}
		return project.Run(cmd, args)
	},
	SilenceUsage: true,
}

func init() {
	newCmd.AddCommand(project.CmdProject)
	rootCmd.AddCommand(newCmd)
}
