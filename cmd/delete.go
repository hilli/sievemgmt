package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var deleteYes bool

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a script from the server",
	Args:  exactArgs(1, "Pass the script name to delete"),
	RunE: func(_ *cobra.Command, args []string) error {
		name := args[0]

		if !deleteYes {
			fmt.Fprintf(os.Stderr, "delete script %q? [y/N]: ", name)
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			if !isYes(line) {
				fmt.Fprintln(os.Stderr, "aborted")
				return nil
			}
		}

		c, err := connect()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.DeleteScript(name); err != nil {
			return fmt.Errorf("deleting script %q: %w", name, err)
		}
		fmt.Fprintf(os.Stderr, "deleted %q\n", name)
		return nil
	},
}

// isYes reports whether the input is an affirmative response.
func isYes(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "y" || s == "yes"
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false,
		"do not prompt for confirmation")
	rootCmd.AddCommand(deleteCmd)
}
