package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hilli/sievemgmt/internal/config"
)

var (
	completionMailboxes = liveCompletionMailboxes
	completionScripts   = liveCompletionScripts
)

func completeAccounts(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return accountCompletionNames(cfg), cobra.ShellCompDirectiveNoFileComp
}

func completeMailboxes(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	boxes, err := completionMailboxes()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return boxes, cobra.ShellCompDirectiveNoFileComp
}

func completeScripts(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	scripts, err := completionScripts()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return scripts, cobra.ShellCompDirectiveNoFileComp
}

func completeFoldersSet(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if foldersGlobal {
		if len(args) == 0 {
			return completeScripts(nil, nil, "")
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	switch len(args) {
	case 0:
		return completeMailboxes(nil, nil, "")
	case 1:
		return completeScripts(nil, nil, "")
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func accountCompletionNames(cfg *config.Config) []string {
	accts := cfg.Accounts()
	names := make([]string, 0, len(accts))
	for _, acct := range accts {
		names = append(names, acct.Name)
	}
	return names
}

func liveCompletionMailboxes() ([]string, error) {
	c, err := connectIMAP()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	boxes, err := c.Mailboxes()
	if err != nil {
		return nil, err
	}
	return boxes, nil
}

func liveCompletionScripts() ([]string, error) {
	c, err := connect()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	scripts, _, err := c.ListScripts()
	if err != nil {
		return nil, fmt.Errorf("listing scripts: %w", err)
	}
	return scripts, nil
}
