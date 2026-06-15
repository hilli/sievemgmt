package imap

import (
	"testing"

	"github.com/hilli/sievemgmt/internal/config"
)

func TestResolveHostPort(t *testing.T) {
	tests := []struct {
		name     string
		acct     config.Account
		hostport string
		host     string
	}{
		{
			name:     "default from sieve server host",
			acct:     config.Account{Name: "a", Server: "mail.example.com"},
			hostport: "mail.example.com:993",
			host:     "mail.example.com",
		},
		{
			name:     "strip sieve port",
			acct:     config.Account{Name: "a", Server: "mail.example.com:4190"},
			hostport: "mail.example.com:993",
			host:     "mail.example.com",
		},
		{
			name:     "explicit imap server",
			acct:     config.Account{Name: "a", Server: "mail.example.com:4190", IMAPServer: "imap.example.com:1993"},
			hostport: "imap.example.com:1993",
			host:     "imap.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hostport, host, err := ResolveHostPort(tt.acct)
			if err != nil {
				t.Fatalf("ResolveHostPort: %v", err)
			}
			if hostport != tt.hostport || host != tt.host {
				t.Fatalf("got %q/%q, want %q/%q", hostport, host, tt.hostport, tt.host)
			}
		})
	}
}

func TestParseListMailbox(t *testing.T) {
	tests := map[string]string{
		`* LIST () "/" Inbox`:                "Inbox",
		`* LIST (\HasNoChildren) "/" "Sent"`: "Sent",
		`* LIST () "/" "Projects/Foo Bar"`:   "Projects/Foo Bar",
	}
	for line, want := range tests {
		got, ok := parseListMailbox(line)
		if !ok || got != want {
			t.Fatalf("parseListMailbox(%q) = %q/%v, want %q/true", line, got, ok, want)
		}
	}
}

func TestParseMetadataValue(t *testing.T) {
	tests := map[string]string{
		`* METADATA Inbox (/shared/imapsieve/script "inbox-events")`:        "inbox-events",
		`* METADATA "Projects/Foo" (/shared/imapsieve/script "foo-events")`: "foo-events",
	}
	for line, want := range tests {
		got, ok := parseMetadataValue(line, IMAPSieveScriptEntry)
		if !ok || got != want {
			t.Fatalf("parseMetadataValue(%q) = %q/%v, want %q/true", line, got, ok, want)
		}
	}
}

func TestQuote(t *testing.T) {
	got := quote(`Projects/"Foo"\Archive`)
	want := `"Projects/\"Foo\"\\Archive"`
	if got != want {
		t.Fatalf("quote = %q, want %q", got, want)
	}
}
