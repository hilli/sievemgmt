// Package config loads and merges sievemgmt account configuration from YAML.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Account holds the connection details for a single mail account.
type Account struct {
	// Name is the config key identifying the account (e.g. "primary").
	Name string `yaml:"-"`
	// Email is the username used for authentication.
	Email string `yaml:"email"`
	// Password is the plaintext password used for authentication.
	Password string `yaml:"password"`
	// Server is the host, optionally with a ":port" suffix.
	Server string `yaml:"server"`
	// IMAPServer is the IMAP TLS host, optionally with a ":port" suffix. If
	// empty, the host from Server is used with port 993.
	IMAPServer string `yaml:"imap_server"`
}

// Config holds all configured accounts in the order they were first seen.
type Config struct {
	accounts []Account
	byName   map[string]int
}

// EnvAccount is the environment variable used to select the active account.
const EnvAccount = "SIEVEMGMT_ACCOUNT"

// configFileName is the base name of the config file in both search locations.
const configFileName = "sievemgmt.yaml"

// SearchPaths returns the config file locations in merge order. Files listed
// later override earlier ones on a per-account-field basis.
func SearchPaths() []string {
	paths := []string{}
	if p := UserConfigPath(); p != "" {
		paths = append(paths, p)
	}
	paths = append(paths, LocalConfigPath())
	return paths
}

// UserConfigPath returns the per-user config path
// (~/.config/sievemgmt/sievemgmt.yaml), or "" if the home dir is unknown.
func UserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "sievemgmt", configFileName)
}

// LocalConfigPath returns the config path in the current directory.
func LocalConfigPath() string {
	return configFileName
}

// Load reads and merges the config from the default search paths. It is not an
// error for individual files to be missing, but at least one must exist.
func Load() (*Config, error) {
	return LoadFromPaths(SearchPaths())
}

// LoadFromPaths reads and merges the config from the given paths in order.
func LoadFromPaths(paths []string) (*Config, error) {
	cfg := &Config{byName: map[string]int{}}
	found := false

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		found = true
		if err := cfg.merge(data, path); err != nil {
			return nil, err
		}
	}

	if !found {
		return nil, fmt.Errorf("no config file found (looked in: %v)", paths)
	}
	if len(cfg.accounts) == 0 {
		return nil, fmt.Errorf("no accounts defined in config")
	}
	return cfg, nil
}

// merge parses a single YAML document and merges its accounts into the config,
// preserving first-seen ordering while letting later files override fields.
func (c *Config) merge(data []byte, path string) error {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	if len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("parsing %s: expected a mapping of account names", path)
	}

	// Mapping nodes store keys and values as alternating children.
	for i := 0; i+1 < len(root.Content); i += 2 {
		name := root.Content[i].Value
		var acct Account
		if err := root.Content[i+1].Decode(&acct); err != nil {
			return fmt.Errorf("parsing account %q in %s: %w", name, path, err)
		}
		acct.Name = name

		if idx, ok := c.byName[name]; ok {
			c.accounts[idx] = mergeAccount(c.accounts[idx], acct)
		} else {
			c.byName[name] = len(c.accounts)
			c.accounts = append(c.accounts, acct)
		}
	}
	return nil
}

// mergeAccount overlays non-empty fields of override onto base.
func mergeAccount(base, override Account) Account {
	if override.Email != "" {
		base.Email = override.Email
	}
	if override.Password != "" {
		base.Password = override.Password
	}
	if override.Server != "" {
		base.Server = override.Server
	}
	if override.IMAPServer != "" {
		base.IMAPServer = override.IMAPServer
	}
	return base
}

// Accounts returns all accounts in first-seen order.
func (c *Config) Accounts() []Account {
	out := make([]Account, len(c.accounts))
	copy(out, c.accounts)
	return out
}

// Lookup returns the account with the given name.
func (c *Config) Lookup(name string) (Account, error) {
	if idx, ok := c.byName[name]; ok {
		return c.accounts[idx], nil
	}
	return Account{}, fmt.Errorf("account %q not found in config", name)
}

// Default returns the first account in the config.
func (c *Config) Default() Account {
	return c.accounts[0]
}

// Select resolves the account to use given an explicit name (from --account or
// the environment). An empty name selects the default (first) account.
func (c *Config) Select(name string) (Account, error) {
	if name == "" {
		return c.Default(), nil
	}
	return c.Lookup(name)
}
