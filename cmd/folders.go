package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	foldersListAll bool
	foldersGlobal  bool
)

var foldersCmd = &cobra.Command{
	Use:     "folders",
	Aliases: []string{"folder"},
	Short:   "Manage IMAPSIEVE script associations for mailboxes",
	Long: `Manage RFC 6785 IMAPSIEVE mailbox bindings.

Scripts are still uploaded with ManageSieve. A mailbox association is stored as
the IMAP METADATA entry /shared/imapsieve/script, whose value is the uploaded
script name. SETACTIVE is not used for IMAPSIEVE associations.`,
}

var foldersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List mailboxes associated with IMAPSIEVE scripts",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		c, err := connectIMAP()
		if err != nil {
			return err
		}
		defer c.Close()

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "MAILBOX\tSCRIPT")
		printed := 0

		if script, ok, err := c.ScriptForMailbox(""); err != nil {
			return fmt.Errorf("reading server-level IMAPSIEVE association: %w", err)
		} else if ok || foldersListAll {
			fmt.Fprintf(w, "%s\t%s\n", "(server)", script)
			printed++
		}

		boxes, err := c.Mailboxes()
		if err != nil {
			return fmt.Errorf("listing mailboxes: %w", err)
		}
		for _, mailbox := range boxes {
			script, ok, err := c.ScriptForMailbox(mailbox)
			if err != nil {
				return fmt.Errorf("reading IMAPSIEVE association for %q: %w", mailbox, err)
			}
			if !ok && !foldersListAll {
				continue
			}
			fmt.Fprintf(w, "%s\t%s\n", mailbox, script)
			printed++
		}
		if printed == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no IMAPSIEVE folder associations")
			return nil
		}
		return w.Flush()
	},
}

var foldersServerListCmd = &cobra.Command{
	Use:     "server-list",
	Aliases: []string{"mailboxes", "ls"},
	Short:   "List IMAP mailboxes available for IMAPSIEVE associations",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		c, err := connectIMAP()
		if err != nil {
			return err
		}
		defer c.Close()

		boxes, err := c.Mailboxes()
		if err != nil {
			return fmt.Errorf("listing mailboxes: %w", err)
		}
		if len(boxes) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "no mailboxes on server")
			return nil
		}
		for _, mailbox := range boxes {
			fmt.Fprintln(cmd.OutOrStdout(), mailbox)
		}
		return nil
	},
}

var foldersSetCmd = &cobra.Command{
	Use:     "set <mailbox> <script> | set --global <script>",
	Aliases: []string{"associate", "assign"},
	Short:   "Associate a mailbox with an IMAPSIEVE script",
	Long: `Associate a mailbox with an uploaded Sieve script for IMAP events.

The script must already exist on the ManageSieve server. Use --global and pass
only <script> to set the server-level fallback association.`,
	Args:              validateFoldersSetArgs,
	ValidArgsFunction: completeFoldersSet,
	RunE: func(cmd *cobra.Command, args []string) error {
		mailbox, script := "", args[0]
		if !foldersGlobal {
			mailbox, script = args[0], args[1]
		}
		if err := ensureScriptExists(script); err != nil {
			return err
		}

		c, err := connectIMAP()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.SetScriptForMailbox(mailbox, script); err != nil {
			return fmt.Errorf("setting IMAPSIEVE association: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "associated %s with %q\n", mailboxLabel(mailbox), script)
		return nil
	},
}

var foldersUnsetCmd = &cobra.Command{
	Use:     "unset <mailbox> | unset --global",
	Aliases: []string{"remove", "rm", "unassociate"},
	Short:   "Remove a mailbox IMAPSIEVE script association",
	Args:    validateFoldersUnsetArgs,
	ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if foldersGlobal || len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeMailboxes(nil, nil, "")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		mailbox := ""
		if !foldersGlobal {
			mailbox = args[0]
		}
		c, err := connectIMAP()
		if err != nil {
			return err
		}
		defer c.Close()

		if err := c.RemoveScriptForMailbox(mailbox); err != nil {
			return fmt.Errorf("removing IMAPSIEVE association: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "removed IMAPSIEVE association for %s\n", mailboxLabel(mailbox))
		return nil
	},
}

func ensureScriptExists(script string) error {
	c, err := connect()
	if err != nil {
		return err
	}
	defer c.Close()

	scripts, _, err := c.ListScripts()
	if err != nil {
		return fmt.Errorf("listing scripts: %w", err)
	}
	for _, name := range scripts {
		if name == script {
			return nil
		}
	}
	return fmt.Errorf("script %q not found; upload it first with `sievemgmt upload`", script)
}

func validateFoldersSetArgs(cmd *cobra.Command, args []string) error {
	if foldersGlobal {
		if len(args) != 1 {
			return fmt.Errorf("usage: %s\n\nWith --global, pass exactly one script name to use as the server-level IMAPSIEVE fallback", cmd.UseLine())
		}
		return nil
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: %s\n\nPass the mailbox name and uploaded script name, or use --global with only a script name", cmd.UseLine())
	}
	return nil
}

func validateFoldersUnsetArgs(cmd *cobra.Command, args []string) error {
	if foldersGlobal {
		if len(args) != 0 {
			return fmt.Errorf("usage: %s\n\nWith --global, do not pass a mailbox name; the server-level fallback association is removed", cmd.UseLine())
		}
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: %s\n\nPass the mailbox name whose IMAPSIEVE association should be removed, or use --global for the server-level fallback", cmd.UseLine())
	}
	return nil
}

func mailboxLabel(mailbox string) string {
	if mailbox == "" {
		return "server-level fallback"
	}
	return fmt.Sprintf("mailbox %q", mailbox)
}

func init() {
	foldersListCmd.Flags().BoolVar(&foldersListAll, "all", false,
		"show all mailboxes, including those without an IMAPSIEVE script")
	foldersSetCmd.Flags().BoolVar(&foldersGlobal, "global", false,
		"set the server-level fallback association instead of a mailbox association")
	foldersUnsetCmd.Flags().BoolVar(&foldersGlobal, "global", false,
		"remove the server-level fallback association instead of a mailbox association")

	foldersCmd.AddCommand(foldersListCmd, foldersServerListCmd, foldersSetCmd, foldersUnsetCmd)
	rootCmd.AddCommand(foldersCmd)
}
