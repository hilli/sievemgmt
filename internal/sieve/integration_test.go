//go:build integration

// Package sieve integration tests run against the live ManageSieve server using
// the accounts listed in tmp/accounts. Run them with:
//
//	go test -tags integration ./internal/sieve/...
//
// The tests are skipped automatically when tmp/accounts is absent. Set
// SIEVE_TEST_SERVER to the ManageSieve server to test against.
package sieve

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hilli/sievemgmt/internal/config"
)

// loadTestAccounts reads tmp/accounts (lines of "email:password") relative to
// the repository root and returns them as config.Account values.
func loadTestAccounts(t *testing.T) []config.Account {
	t.Helper()

	server := os.Getenv("SIEVE_TEST_SERVER")
	if server == "" {
		t.Skip("SIEVE_TEST_SERVER not set; skipping integration tests")
	}

	// Walk up to find tmp/accounts so the test works from the package dir.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	var path string
	for {
		candidate := filepath.Join(dir, "tmp", "accounts")
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if path == "" {
		t.Skip("tmp/accounts not found; skipping integration tests")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	defer f.Close()

	var accounts []config.Account
	scanner := bufio.NewScanner(f)
	n := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		n++
		accounts = append(accounts, config.Account{
			Name:     line[:idx],
			Email:    line[:idx],
			Password: line[idx+1:],
			Server:   server,
		})
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(accounts) == 0 {
		t.Skip("tmp/accounts contained no usable accounts")
	}
	return accounts
}

// TestIntegrationConnect verifies that each account can authenticate and list.
func TestIntegrationConnect(t *testing.T) {
	for _, acct := range loadTestAccounts(t) {
		acct := acct
		t.Run(acct.Email, func(t *testing.T) {
			c, err := Connect(acct)
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			defer c.Close()

			if _, _, err := c.ListScripts(); err != nil {
				t.Fatalf("list scripts: %v", err)
			}
		})
	}
}

// TestIntegrationScriptLifecycle exercises check/put/get/activate/rename/delete
// on a uniquely named temporary script, restoring the original active script.
func TestIntegrationScriptLifecycle(t *testing.T) {
	const (
		tmpName     = "sievemgmt_integration_test"
		renamedName = "sievemgmt_integration_test_renamed"
		script      = "# sievemgmt integration test\nkeep;\n"
		updated     = "# sievemgmt integration test (updated)\nstop;\n"
	)

	for _, acct := range loadTestAccounts(t) {
		acct := acct
		t.Run(acct.Email, func(t *testing.T) {
			c, err := Connect(acct)
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			defer c.Close()

			_, originalActive, err := c.ListScripts()
			if err != nil {
				t.Fatalf("list scripts: %v", err)
			}

			// Cleanup: remove test scripts and restore active script.
			defer func() {
				_ = c.DeleteScript(tmpName)
				_ = c.DeleteScript(renamedName)
				if originalActive != "" {
					_ = c.ActivateScript(originalActive)
				}
			}()

			if _, err := c.CheckScript(script); err != nil {
				t.Fatalf("check script: %v", err)
			}

			if _, err := c.PutScript(tmpName, script); err != nil {
				t.Fatalf("put script: %v", err)
			}

			got, err := c.GetScript(tmpName)
			if err != nil {
				t.Fatalf("get script: %v", err)
			}
			if got != script {
				t.Fatalf("round-trip mismatch:\n got: %q\nwant: %q", got, script)
			}

			// Overwrite with updated content.
			if _, err := c.PutScript(tmpName, updated); err != nil {
				t.Fatalf("update script: %v", err)
			}
			got, err = c.GetScript(tmpName)
			if err != nil {
				t.Fatalf("get updated script: %v", err)
			}
			if got != updated {
				t.Fatalf("update mismatch:\n got: %q\nwant: %q", got, updated)
			}

			if err := c.ActivateScript(tmpName); err != nil {
				t.Fatalf("activate script: %v", err)
			}
			if _, active, err := c.ListScripts(); err != nil {
				t.Fatalf("list after activate: %v", err)
			} else if active != tmpName {
				t.Fatalf("active script = %q, want %q", active, tmpName)
			}

			// Deactivate so the script can be renamed/deleted.
			if err := c.ActivateScript(""); err != nil {
				t.Fatalf("deactivate: %v", err)
			}

			if err := c.RenameScript(tmpName, renamedName); err != nil {
				t.Fatalf("rename script: %v", err)
			}

			if err := c.DeleteScript(renamedName); err != nil {
				t.Fatalf("delete script: %v", err)
			}
		})
	}
}
