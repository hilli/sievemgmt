// Package tlsconfig builds TLS client configs shared by protocol clients.
package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// EnvCAFile points at a PEM CA bundle to trust in addition to system roots.
const EnvCAFile = "SIEVEMGMT_TLS_CA_FILE"

// Client returns a TLS client config for serverName.
func Client(serverName string) (*tls.Config, error) {
	cfg := &tls.Config{ServerName: serverName}
	caFile := os.Getenv(EnvCAFile)
	if caFile == "" {
		return cfg, nil
	}

	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", EnvCAFile, err)
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("loading system certificate pool: %w", err)
	}
	if roots == nil {
		roots = x509.NewCertPool()
	}
	if !roots.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("loading %s: no certificates found in %s", EnvCAFile, caFile)
	}
	cfg.RootCAs = roots
	return cfg, nil
}
