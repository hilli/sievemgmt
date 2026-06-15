package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a script on the server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName, newName := args[0], args[1]

		c, err := connect()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.RenameScript(oldName, newName); err != nil {
			return fmt.Errorf("renaming script %q to %q: %w", oldName, newName, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "renamed %q to %q\n", oldName, newName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
