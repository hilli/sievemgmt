package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check <file>",
	Short: "Check the validity of a local script against the server",
	Args:  exactArgs(1, "Pass the local .sieve file to validate"),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file, err)
		}

		c, err := connect()
		if err != nil {
			return err
		}
		defer c.Close()

		warnings, err := c.CheckScript(string(data))
		if err != nil {
			return fmt.Errorf("script is not valid: %w", err)
		}
		if warnings != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "valid, with warnings:\n%s\n", warnings)
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "valid")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
