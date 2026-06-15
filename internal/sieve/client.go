// Package sieve provides a thin connection helper over the managesieve client.
package sieve

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"go.guido-berhoerster.org/managesieve"

	"github.com/hilli/sievemgmt/internal/config"
)

// DefaultPort is the standard ManageSieve port (RFC 5804).
const DefaultPort = "4190"

// DialTimeout bounds how long Connect waits for the initial TCP connection.
var DialTimeout = 15 * time.Second

// Client wraps a managesieve.Client together with the host it is connected to.
type Client struct {
	*managesieve.Client
	host string
}

// ResolveHostPort splits an account server value into a "host:port" address and
// the bare host name. If no port is given, an SRV lookup is attempted, falling
// back to the default ManageSieve port.
func ResolveHostPort(server string) (hostport, host string) {
	server = strings.TrimSpace(server)
	if h, p, err := net.SplitHostPort(server); err == nil {
		return net.JoinHostPort(h, p), h
	}

	host = server
	if services, err := managesieve.LookupService(host); err == nil && len(services) > 0 {
		return services[0], host
	}
	return net.JoinHostPort(host, DefaultPort), host
}

// Connect dials the account's server, negotiates STARTTLS and authenticates
// using the PLAIN SASL mechanism.
func Connect(acct config.Account) (*Client, error) {
	if acct.Server == "" {
		return nil, fmt.Errorf("account %q has no server configured", acct.Name)
	}
	if acct.Email == "" {
		return nil, fmt.Errorf("account %q has no email configured", acct.Name)
	}

	hostport, host := ResolveHostPort(acct.Server)

	conn, err := net.DialTimeout("tcp", hostport, DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", hostport, err)
	}

	c, err := managesieve.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("initializing ManageSieve session with %s: %w", host, err)
	}

	if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("starting TLS with %s: %w", host, err)
	}

	auth := managesieve.PlainAuth("", acct.Email, acct.Password, host)
	if err := c.Authenticate(auth); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("authenticating as %s: %w", acct.Email, err)
	}

	return &Client{Client: c, host: host}, nil
}

// Host returns the bare host name the client is connected to.
func (c *Client) Host() string { return c.host }

// Close logs out and closes the underlying connection.
func (c *Client) Close() error {
	if err := c.Logout(); err != nil {
		_ = c.Client.Close()
		return err
	}
	return nil
}
