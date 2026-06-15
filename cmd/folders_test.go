package cmd

import (
	"strings"
	"testing"
)

func TestFoldersServerListRegistered(t *testing.T) {
	cmd, _, err := foldersCmd.Find([]string{"server-list"})
	if err != nil {
		t.Fatalf("Find server-list: %v", err)
	}
	if cmd != foldersServerListCmd {
		t.Fatalf("Find server-list = %q, want foldersServerListCmd", cmd.Name())
	}
	if !strings.Contains(cmd.Short, "List IMAP mailboxes") {
		t.Fatalf("server-list short help = %q", cmd.Short)
	}
}

func TestValidateFoldersSetArgs(t *testing.T) {
	t.Cleanup(func() { foldersGlobal = false })

	foldersGlobal = false
	if err := validateFoldersSetArgs(foldersSetCmd, nil); err == nil || !strings.Contains(err.Error(), "Pass the mailbox name and uploaded script name") {
		t.Fatalf("set without args error = %v", err)
	}
	if err := validateFoldersSetArgs(foldersSetCmd, []string{"Inbox", "script"}); err != nil {
		t.Fatalf("set with mailbox/script: %v", err)
	}

	foldersGlobal = true
	if err := validateFoldersSetArgs(foldersSetCmd, nil); err == nil || !strings.Contains(err.Error(), "pass exactly one script name") {
		t.Fatalf("set --global without script error = %v", err)
	}
	if err := validateFoldersSetArgs(foldersSetCmd, []string{"script"}); err != nil {
		t.Fatalf("set --global with script: %v", err)
	}
}

func TestValidateFoldersUnsetArgs(t *testing.T) {
	t.Cleanup(func() { foldersGlobal = false })

	foldersGlobal = false
	if err := validateFoldersUnsetArgs(foldersUnsetCmd, nil); err == nil || !strings.Contains(err.Error(), "Pass the mailbox name") {
		t.Fatalf("unset without args error = %v", err)
	}
	if err := validateFoldersUnsetArgs(foldersUnsetCmd, []string{"Inbox"}); err != nil {
		t.Fatalf("unset with mailbox: %v", err)
	}

	foldersGlobal = true
	if err := validateFoldersUnsetArgs(foldersUnsetCmd, []string{"Inbox"}); err == nil || !strings.Contains(err.Error(), "do not pass a mailbox name") {
		t.Fatalf("unset --global with mailbox error = %v", err)
	}
	if err := validateFoldersUnsetArgs(foldersUnsetCmd, nil); err != nil {
		t.Fatalf("unset --global without args: %v", err)
	}
}
