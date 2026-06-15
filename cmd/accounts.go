package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List configured accounts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		selected := selectedAccountName()
		defaultName := cfg.Default().Name

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "\tNAME\tEMAIL\tSERVER")
		for _, a := range cfg.Accounts() {
			marker := " "
			if (selected == "" && a.Name == defaultName) || a.Name == selected {
				marker = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", marker, a.Name, a.Email, a.Server)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(accountsCmd)
}
