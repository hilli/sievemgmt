package cmd

import (
	"strings"
	"testing"
)

func TestCommonArgErrorsAreHelpful(t *testing.T) {
	tests := []struct {
		name    string
		argsErr error
		want    string
	}{
		{
			name:    "check missing file",
			argsErr: checkCmd.Args(checkCmd, nil),
			want:    "Pass the local .sieve file",
		},
		{
			name:    "delete missing name",
			argsErr: deleteCmd.Args(deleteCmd, nil),
			want:    "Pass the script name to delete",
		},
		{
			name:    "rename missing names",
			argsErr: renameCmd.Args(renameCmd, []string{"old"}),
			want:    "Pass the current script name and the new script name",
		},
		{
			name:    "upload missing file",
			argsErr: uploadCmd.Args(uploadCmd, nil),
			want:    "Pass the local .sieve file",
		},
		{
			name:    "download too many names",
			argsErr: downloadCmd.Args(downloadCmd, []string{"one", "two"}),
			want:    "Pass at most one script name",
		},
		{
			name:    "edit too many names",
			argsErr: editCmd.Args(editCmd, []string{"one", "two"}),
			want:    "Pass at most one script name",
		},
		{
			name:    "account add missing name",
			argsErr: accountAddCmd.Args(accountAddCmd, nil),
			want:    "Pass the account name",
		},
		{
			name:    "account remove missing name",
			argsErr: accountRemoveCmd.Args(accountRemoveCmd, nil),
			want:    "Pass the account name to remove",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.argsErr == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(tt.argsErr.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", tt.argsErr, tt.want)
			}
			if !strings.Contains(tt.argsErr.Error(), "usage: ") {
				t.Fatalf("error = %q, want usage line", tt.argsErr)
			}
		})
	}
}

func TestActivateArgErrors(t *testing.T) {
	t.Cleanup(func() { activateNone = false })

	activateNone = false
	if err := activateCmd.Args(activateCmd, []string{"one", "two"}); err == nil || !strings.Contains(err.Error(), "Pass one script name") {
		t.Fatalf("activate too many args error = %v", err)
	}
	if err := activateCmd.Args(activateCmd, []string{"script"}); err != nil {
		t.Fatalf("activate with script: %v", err)
	}

	activateNone = true
	if err := activateCmd.Args(activateCmd, []string{"script"}); err == nil || !strings.Contains(err.Error(), "Use either --none or a script name") {
		t.Fatalf("activate --none with script error = %v", err)
	}
	if err := activateCmd.Args(activateCmd, nil); err != nil {
		t.Fatalf("activate --none without args: %v", err)
	}
}
