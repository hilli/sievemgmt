package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [name]",
	Short: "Edit a script in $EDITOR and upload it",
	Long: `Download a script (the active one if no name is given) to a temporary
file, open it in $EDITOR, then check and upload it when the editor exits. If the
server rejects the script you are asked whether to edit again or save a local
copy instead.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		c, err := connect()
		if err != nil {
			return err
		}
		defer c.Close()

		scripts, active, err := c.ListScripts()
		if err != nil {
			return fmt.Errorf("listing scripts: %w", err)
		}

		name := active
		if len(args) == 1 {
			name = args[0]
		}
		if name == "" {
			return fmt.Errorf("no active script; specify a script name to edit")
		}

		existing := contains(scripts, name)
		content := ""
		if existing {
			content, err = c.GetScript(name)
			if err != nil {
				return fmt.Errorf("downloading script %q: %w", name, err)
			}
		}

		tmp, err := os.CreateTemp("", "sievemgmt-*.sieve")
		if err != nil {
			return fmt.Errorf("creating temp file: %w", err)
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath)
		if _, err := tmp.WriteString(content); err != nil {
			tmp.Close()
			return fmt.Errorf("writing temp file: %w", err)
		}
		tmp.Close()

		reader := bufio.NewReader(os.Stdin)
		for {
			if err := runEditor(tmpPath); err != nil {
				return err
			}

			data, err := os.ReadFile(tmpPath)
			if err != nil {
				return fmt.Errorf("reading edited file: %w", err)
			}
			edited := string(data)

			warnings, checkErr := c.CheckScript(edited)
			if checkErr == nil {
				if warnings != "" {
					fmt.Fprintf(os.Stderr, "warning: %s\n", warnings)
				}
				if _, err := c.PutScript(name, edited); err != nil {
					return fmt.Errorf("uploading script %q: %w", name, err)
				}
				fmt.Fprintf(os.Stderr, "uploaded %q\n", name)
				return nil
			}

			fmt.Fprintf(os.Stderr, "script check failed: %s\n", checkErr)
			fmt.Fprint(os.Stderr, "edit again or save locally? [e/s]: ")
			answer, _ := reader.ReadString('\n')
			switch strings.ToLower(strings.TrimSpace(answer)) {
			case "s", "save", "save locally":
				return saveLocal(name, edited)
			default:
				// loop and re-open the editor
			}
		}
	},
}

// runEditor opens path in the user's editor and waits for it to exit.
func runEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		for _, candidate := range []string{"vim", "vi", "nano"} {
			if _, err := exec.LookPath(candidate); err == nil {
				editor = candidate
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found; set $EDITOR")
	}

	// Support editors invoked with arguments (e.g. "code --wait").
	fields := strings.Fields(editor)
	fields = append(fields, path)
	c := exec.Command(fields[0], fields[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}
	return nil
}

// saveLocal writes content to ./<name>.sieve and reports the path.
func saveLocal(name, content string) error {
	fileName := name
	if filepath.Ext(fileName) == "" {
		fileName += ".sieve"
	}
	if err := os.WriteFile(fileName, []byte(content), 0o600); err != nil {
		return fmt.Errorf("saving local copy: %w", err)
	}
	abs, _ := filepath.Abs(fileName)
	fmt.Fprintf(os.Stderr, "saved local copy to %s\n", abs)
	return nil
}

// contains reports whether s is present in the slice.
func contains(items []string, s string) bool {
	for _, item := range items {
		if item == s {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(editCmd)
}
