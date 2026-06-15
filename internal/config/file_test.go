package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileAddListRemoveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "sievemgmt.yaml")

	// Start from a non-existent file.
	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if names := f.Names(); len(names) != 0 {
		t.Fatalf("expected empty file, got %v", names)
	}

	if err := f.Set(Account{Name: "primary", Email: "p@example.com", Password: "pw", Server: "mail.example.com"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := f.Set(Account{Name: "work", Email: "w@example.com", Password: "wpw", Server: "mail.example.com:4190", IMAPServer: "imap.example.com:993"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := f.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload and verify order + contents.
	f2, err := LoadFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	accts, err := f2.Accounts()
	if err != nil {
		t.Fatalf("Accounts: %v", err)
	}
	if len(accts) != 2 || accts[0].Name != "primary" || accts[1].Name != "work" {
		t.Fatalf("unexpected accounts/order: %+v", accts)
	}
	if accts[1].Server != "mail.example.com:4190" {
		t.Errorf("work server = %q", accts[1].Server)
	}
	if accts[1].IMAPServer != "imap.example.com:993" {
		t.Errorf("work imap server = %q", accts[1].IMAPServer)
	}

	// File permissions must be owner-only.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file perm = %o, want 600", perm)
	}

	// Update existing account in place (order preserved).
	if !f2.Has("primary") {
		t.Fatal("Has(primary) = false")
	}
	if err := f2.Set(Account{Name: "primary", Email: "new@example.com", Password: "np", Server: "new.example.com"}); err != nil {
		t.Fatalf("Set update: %v", err)
	}
	accts, _ = f2.Accounts()
	if accts[0].Name != "primary" || accts[0].Email != "new@example.com" {
		t.Fatalf("update failed: %+v", accts[0])
	}

	// Remove.
	if !f2.Remove("primary") {
		t.Fatal("Remove(primary) = false")
	}
	if f2.Remove("missing") {
		t.Fatal("Remove(missing) = true")
	}
	if names := f2.Names(); len(names) != 1 || names[0] != "work" {
		t.Fatalf("after remove: %v", names)
	}
}

func TestFilePreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sievemgmt.yaml")
	content := "# top comment\nprimary:\n  email: p@example.com # inline\n  password: pw\n  server: s\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	f, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if err := f.Set(Account{Name: "work", Email: "w@example.com", Password: "wp", Server: "s2"}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := f.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !contains(string(data), "# top comment") {
		t.Errorf("top comment not preserved:\n%s", data)
	}
}

// contains is a tiny helper to avoid importing strings for one check.
func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
