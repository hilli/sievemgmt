package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var uploadActivate bool

var uploadCmd = &cobra.Command{
	Use:   "upload <file> [name]",
	Short: "Upload a script from a local file",
	Long: `Upload a Sieve script from a local file. The script is checked by the
server before being stored. If no name is given, the file's base name without
the .sieve extension is used. Use --activate to make it the active script.`,
	Args: rangeArgs(1, 2, "Pass the local .sieve file and optionally the server-side script name"),
	RunE: func(_ *cobra.Command, args []string) error {
		file := args[0]
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file, err)
		}
		content := string(data)

		name := ""
		if len(args) == 2 {
			name = args[1]
		} else {
			base := filepath.Base(file)
			name = strings.TrimSuffix(base, filepath.Ext(base))
		}
		if name == "" {
			return fmt.Errorf("could not determine script name; pass one explicitly")
		}

		c, err := connect()
		if err != nil {
			return err
		}
		defer c.Close()

		if warnings, err := c.CheckScript(content); err != nil {
			return fmt.Errorf("script check failed: %w", err)
		} else if warnings != "" {
			fmt.Fprintf(os.Stderr, "warning: %s\n", warnings)
		}

		if warnings, err := c.PutScript(name, content); err != nil {
			return fmt.Errorf("uploading script %q: %w", name, err)
		} else if warnings != "" {
			fmt.Fprintf(os.Stderr, "warning: %s\n", warnings)
		}
		fmt.Fprintf(os.Stderr, "uploaded %q\n", name)

		if uploadActivate {
			if err := c.ActivateScript(name); err != nil {
				return fmt.Errorf("activating script %q: %w", name, err)
			}
			fmt.Fprintf(os.Stderr, "activated %q\n", name)
		}
		return nil
	},
}

func init() {
	uploadCmd.Flags().BoolVar(&uploadActivate, "activate", false,
		"activate the script after uploading")
	rootCmd.AddCommand(uploadCmd)
}
