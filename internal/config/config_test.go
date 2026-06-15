package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

func TestLoadSingleFileOrderPreserved(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.yaml", `
zeta:
  email: zeta@example.com
  password: z
  server: mail.example.com
alpha:
  email: alpha@example.com
  password: a
  server: mail.example.com
`)

	cfg, err := LoadFromPaths([]string{path})
	if err != nil {
		t.Fatalf("LoadFromPaths: %v", err)
	}

	accts := cfg.Accounts()
	if len(accts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accts))
	}
	if accts[0].Name != "zeta" || accts[1].Name != "alpha" {
		t.Fatalf("order not preserved: %q, %q", accts[0].Name, accts[1].Name)
	}
	if cfg.Default().Name != "zeta" {
		t.Fatalf("default should be first account, got %q", cfg.Default().Name)
	}
}

func TestMergeLatterOverrides(t *testing.T) {
	dir := t.TempDir()
	base := writeFile(t, dir, "base.yaml", `
primary:
  email: old@example.com
  password: oldpw
  server: old.example.com
secondary:
  email: sec@example.com
  password: secpw
  server: sec.example.com
`)
	override := writeFile(t, dir, "override.yaml", `
primary:
  password: newpw
  imap_server: imap.example.com:993
third:
  email: third@example.com
  password: thirdpw
  server: third.example.com
`)

	cfg, err := LoadFromPaths([]string{base, override})
	if err != nil {
		t.Fatalf("LoadFromPaths: %v", err)
	}

	primary, err := cfg.Lookup("primary")
	if err != nil {
		t.Fatalf("Lookup primary: %v", err)
	}
	if primary.Password != "newpw" {
		t.Errorf("password not overridden: %q", primary.Password)
	}
	if primary.Email != "old@example.com" {
		t.Errorf("unrelated field changed: %q", primary.Email)
	}
	if primary.IMAPServer != "imap.example.com:993" {
		t.Errorf("imap server not overridden: %q", primary.IMAPServer)
	}

	accts := cfg.Accounts()
	if len(accts) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(accts))
	}
	// Original ordering preserved; newly introduced account appended.
	want := []string{"primary", "secondary", "third"}
	for i, w := range want {
		if accts[i].Name != w {
			t.Errorf("account %d: want %q, got %q", i, w, accts[i].Name)
		}
	}
	if cfg.Default().Name != "primary" {
		t.Errorf("default should remain first account, got %q", cfg.Default().Name)
	}
}

func TestLoadMissingFileSkipped(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.yaml", "primary:\n  email: a@example.com\n  password: a\n  server: s\n")

	cfg, err := LoadFromPaths([]string{filepath.Join(dir, "nope.yaml"), path})
	if err != nil {
		t.Fatalf("LoadFromPaths: %v", err)
	}
	if len(cfg.Accounts()) != 1 {
		t.Fatalf("expected 1 account, got %d", len(cfg.Accounts()))
	}
}

func TestLoadNoFilesIsError(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadFromPaths([]string{filepath.Join(dir, "missing.yaml")}); err == nil {
		t.Fatal("expected error when no config files exist")
	}
}

func TestSelect(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.yaml", `
primary:
  email: p@example.com
  password: p
  server: s
other:
  email: o@example.com
  password: o
  server: s
`)
	cfg, err := LoadFromPaths([]string{path})
	if err != nil {
		t.Fatalf("LoadFromPaths: %v", err)
	}

	if a, err := cfg.Select(""); err != nil || a.Name != "primary" {
		t.Errorf("empty select should be default: %v, %q", err, a.Name)
	}
	if a, err := cfg.Select("other"); err != nil || a.Name != "other" {
		t.Errorf("named select failed: %v, %q", err, a.Name)
	}
	if _, err := cfg.Select("missing"); err == nil {
		t.Error("expected error for missing account")
	}
}
