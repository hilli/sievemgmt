package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var downloadOutput string

var downloadCmd = &cobra.Command{
	Use:   "download [name]",
	Short: "Download a script (defaults to the active script)",
	Long: `Download a script from the server. If no name is given, the currently
active script is downloaded. Use -o to write to a file ("-" means stdout).`,
	Args: maxArgs(1, "Pass at most one script name; omit it to download the active script"),
	RunE: func(_ *cobra.Command, args []string) error {
		c, err := connect()
		if err != nil {
			return err
		}
		defer c.Close()

		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		if name == "" {
			_, active, err := c.ListScripts()
			if err != nil {
				return fmt.Errorf("listing scripts: %w", err)
			}
			if active == "" {
				return fmt.Errorf("no active script to download; specify a name")
			}
			name = active
		}

		content, err := c.GetScript(name)
		if err != nil {
			return fmt.Errorf("downloading script %q: %w", name, err)
		}

		if downloadOutput == "" || downloadOutput == "-" {
			fmt.Print(content)
			return nil
		}
		if err := os.WriteFile(downloadOutput, []byte(content), 0o600); err != nil {
			return fmt.Errorf("writing %s: %w", downloadOutput, err)
		}
		fmt.Fprintf(os.Stderr, "wrote %q to %s\n", name, downloadOutput)
		return nil
	},
}

func init() {
	downloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "",
		`output file ("-" or empty for stdout)`)
	rootCmd.AddCommand(downloadCmd)
}
