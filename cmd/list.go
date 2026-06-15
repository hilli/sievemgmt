package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List scripts on the server, marking the active one",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		c, err := connect()
		if err != nil {
			return err
		}
		defer c.Close()

		scripts, active, err := c.ListScripts()
		if err != nil {
			return fmt.Errorf("listing scripts: %w", err)
		}
		if len(scripts) == 0 {
			fmt.Println("no scripts on server")
			return nil
		}
		for _, name := range scripts {
			marker := "  "
			if name == active {
				marker = "* "
			}
			fmt.Printf("%s%s\n", marker, name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
