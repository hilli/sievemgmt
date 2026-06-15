package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var activateNone bool

var activateCmd = &cobra.Command{
	Use:   "activate [name]",
	Short: "Set the active script (or deactivate all with --none)",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return usageError(cmd, "Pass one script name to activate, or use --none to deactivate all scripts")
		}
		if activateNone && len(args) > 0 {
			return usageError(cmd, "Use either --none or a script name, not both")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		if !activateNone && name == "" {
			return fmt.Errorf("provide a script name or use --none to deactivate")
		}
		if activateNone {
			name = ""
		}

		c, err := connect()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.ActivateScript(name); err != nil {
			return fmt.Errorf("activating script: %w", err)
		}
		if name == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "deactivated all scripts")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "activated %q\n", name)
		}
		return nil
	},
}

func init() {
	activateCmd.Flags().BoolVar(&activateNone, "none", false,
		"deactivate all scripts (no active script)")
	rootCmd.AddCommand(activateCmd)
}
