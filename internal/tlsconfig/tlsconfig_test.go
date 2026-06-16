package tlsconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClientWithoutCAFile(t *testing.T) {
	t.Setenv(EnvCAFile, "")

	cfg, err := Client("mail.example.com")
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	if cfg.ServerName != "mail.example.com" {
		t.Fatalf("ServerName = %q", cfg.ServerName)
	}
	if cfg.RootCAs != nil {
		t.Fatalf("RootCAs set without %s", EnvCAFile)
	}
}

func TestClientRejectsInvalidCAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0600); err != nil {
		t.Fatalf("writing ca file: %v", err)
	}
	t.Setenv(EnvCAFile, path)

	if _, err := Client("mail.example.com"); err == nil {
		t.Fatalf("Client succeeded with invalid %s", EnvCAFile)
	}
}
