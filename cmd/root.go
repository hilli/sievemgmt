// Package cmd implements the sievemgmt CLI commands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hilli/sievemgmt/internal/config"
	"github.com/hilli/sievemgmt/internal/sieve"
)

// accountFlag holds the value of the persistent --account flag.
var accountFlag string

// Build information, overridden at release time via -ldflags.
var (
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"
)

// rootCmd is the base command for the CLI.
var rootCmd = &cobra.Command{
	Use:   "sievemgmt",
	Short: "Manage Sieve scripts on a remote ManageSieve server",
	Long: `sievemgmt manages Sieve scripts on a remote server over the ManageSieve
protocol (RFC 5804). Accounts are configured in YAML and selected with the
--account flag or the SIEVEMGMT_ACCOUNT environment variable.`,
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&accountFlag, "account", "a", "",
		"account to use (overrides "+config.EnvAccount+"; defaults to the first configured account)")
	rootCmd.SetVersionTemplate(
		"sievemgmt {{.Version}} (commit " + GitCommit + ", built " + BuildDate + ")\n")
}

// loadConfig loads the merged configuration.
func loadConfig() (*config.Config, error) {
	return config.Load()
}

// selectedAccountName resolves the account name from the flag or environment.
func selectedAccountName() string {
	if accountFlag != "" {
		return accountFlag
	}
	return os.Getenv(config.EnvAccount)
}

// resolveAccount loads the config and selects the active account.
func resolveAccount() (config.Account, error) {
	cfg, err := loadConfig()
	if err != nil {
		return config.Account{}, err
	}
	return cfg.Select(selectedAccountName())
}

// connect resolves the active account and opens an authenticated connection.
func connect() (*sieve.Client, error) {
	acct, err := resolveAccount()
	if err != nil {
		return nil, err
	}
	return sieve.Connect(acct)
}
