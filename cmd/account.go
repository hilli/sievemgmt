package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/hilli/sievemgmt/internal/config"
)

var (
	accountFile  string
	accountLocal bool

	addEmail    string
	addPassword string
	addServer   string
	addIMAP     string
	addForce    bool
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage accounts in the config file (add, list, remove)",
	Long: `Add, list and remove accounts in a sievemgmt config file.

By default the per-user config (~/.config/sievemgmt/sievemgmt.yaml) is edited.
Use --local to edit ./sievemgmt.yaml, or --file to target a specific path.`,
}

// targetFile resolves which config file the account subcommands operate on.
func targetFile() (*config.File, error) {
	path := accountFile
	if path == "" {
		if accountLocal {
			path = config.LocalConfigPath()
		} else {
			path = config.UserConfigPath()
			if path == "" {
				return nil, fmt.Errorf("cannot determine home directory; use --file or --local")
			}
		}
	}
	return config.LoadFile(path)
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List accounts defined in the config file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		f, err := targetFile()
		if err != nil {
			return err
		}
		accts, err := f.Accounts()
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "# %s\n", f.Path())
		if len(accts) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no accounts")
			return nil
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tEMAIL\tSERVER\tIMAP_SERVER")
		for _, a := range accts {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Name, a.Email, a.Server, a.IMAPServer)
		}
		return w.Flush()
	},
}

var accountAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add (or update) an account in the config file",
	Args:  exactArgs(1, "Pass the account name to add or update"),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		f, err := targetFile()
		if err != nil {
			return err
		}
		if f.Has(name) && !addForce {
			return fmt.Errorf("account %q already exists in %s (use --force to overwrite)", name, f.Path())
		}

		if addEmail == "" {
			return usageError(cmd, "--email is required, e.g. --email you@example.com")
		}
		if addServer == "" {
			return usageError(cmd, `--server is required, e.g. --server mail.example.com or --server mail.example.com:4190`)
		}

		password := addPassword
		if password == "" {
			password, err = promptPassword()
			if err != nil {
				return err
			}
		}

		if err := f.Set(config.Account{
			Name:       name,
			Email:      addEmail,
			Password:   password,
			Server:     addServer,
			IMAPServer: addIMAP,
		}); err != nil {
			return err
		}
		if err := f.Save(); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "saved account %q to %s\n", name, f.Path())
		return nil
	},
}

var accountRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove an account from the config file",
	Args:    exactArgs(1, "Pass the account name to remove"),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		f, err := targetFile()
		if err != nil {
			return err
		}
		if !f.Remove(name) {
			return fmt.Errorf("account %q not found in %s", name, f.Path())
		}
		if err := f.Save(); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "removed account %q from %s\n", name, f.Path())
		return nil
	},
}

// promptPassword reads a password from the terminal without echo, asking twice
// to guard against typos. It falls back to a plain read when stdin is not a TTY.
func promptPassword() (string, error) {
	fd := int(os.Stdin.Fd()) //nolint:gosec // stdin file descriptor always fits in int
	if !term.IsTerminal(fd) {
		var pw string
		if _, err := fmt.Fscanln(os.Stdin, &pw); err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		return pw, nil
	}

	fmt.Fprint(os.Stderr, "Password: ")
	first, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	fmt.Fprint(os.Stderr, "Confirm password: ")
	second, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading password: %w", err)
	}
	if string(first) != string(second) {
		return "", fmt.Errorf("passwords do not match")
	}
	return string(first), nil
}

func init() {
	accountCmd.PersistentFlags().StringVar(&accountFile, "file", "",
		"config file to edit (defaults to the per-user config)")
	accountCmd.PersistentFlags().BoolVar(&accountLocal, "local", false,
		"edit ./sievemgmt.yaml instead of the per-user config")

	accountAddCmd.Flags().StringVar(&addEmail, "email", "", "account email/username (required)")
	accountAddCmd.Flags().StringVar(&addPassword, "password", "", "account password (prompted if omitted)")
	accountAddCmd.Flags().StringVar(&addServer, "server", "", `server host, optional ":port" (required)`)
	accountAddCmd.Flags().StringVar(&addIMAP, "imap-server", "",
		`IMAPS host, optional ":port" (defaults to --server host with port 993)`)
	accountAddCmd.Flags().BoolVar(&addForce, "force", false, "overwrite an existing account")

	accountCmd.AddCommand(accountListCmd, accountAddCmd, accountRemoveCmd)
	rootCmd.AddCommand(accountCmd)
}
