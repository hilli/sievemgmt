package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	"github.com/hilli/sievemgmt/internal/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sievemgmt.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return path
}

func TestAccountCompletionNames(t *testing.T) {
	cfg, err := config.LoadFromPaths([]string{writeConfig(t, `
primary:
  email: p@example.com
  password: p
  server: mail.example.com
work:
  email: w@example.com
  password: w
  server: mail.example.com
`)})
	if err != nil {
		t.Fatalf("LoadFromPaths: %v", err)
	}
	got := accountCompletionNames(cfg)
	want := []string{"primary", "work"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("accountCompletionNames = %v, want %v", got, want)
	}
}

func TestFoldersSetCompletionDirective(t *testing.T) {
	origMailboxes := completionMailboxes
	origScripts := completionScripts
	t.Cleanup(func() {
		foldersGlobal = false
		completionMailboxes = origMailboxes
		completionScripts = origScripts
	})

	completionMailboxes = func() ([]string, error) {
		return []string{"Inbox", "Projects/Foo"}, nil
	}
	completionScripts = func() ([]string, error) {
		return []string{"imap-events", "archive"}, nil
	}

	foldersGlobal = false
	got, directive := foldersSetCmd.ValidArgsFunction(foldersSetCmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("mailbox completion directive = %v, want NoFileComp", directive)
	}
	if want := []string{"Inbox", "Projects/Foo"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mailbox completions = %v, want %v", got, want)
	}

	got, directive = foldersSetCmd.ValidArgsFunction(foldersSetCmd, []string{"Inbox"}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("script completion directive = %v, want NoFileComp", directive)
	}
	if want := []string{"imap-events", "archive"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("script completions = %v, want %v", got, want)
	}

	if _, directive := foldersSetCmd.ValidArgsFunction(foldersSetCmd, []string{"Inbox", "script"}, ""); directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completion directive = %v, want NoFileComp", directive)
	}

	foldersGlobal = true
	got, directive = foldersSetCmd.ValidArgsFunction(foldersSetCmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("global script completion directive = %v, want NoFileComp", directive)
	}
	if want := []string{"imap-events", "archive"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("global script completions = %v, want %v", got, want)
	}
	if _, directive := foldersSetCmd.ValidArgsFunction(foldersSetCmd, []string{"script"}, ""); directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("global completion directive = %v, want NoFileComp", directive)
	}
}
