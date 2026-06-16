// Package imap provides the small subset of IMAP needed for IMAPSIEVE metadata.
package imap

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/hilli/sievemgmt/internal/config"
	"github.com/hilli/sievemgmt/internal/tlsconfig"
)

// IMAPSieveScriptEntry is the RFC 6785 IMAP METADATA entry that selects the
// script for IMAP events on a mailbox or globally.
const IMAPSieveScriptEntry = "/shared/imapsieve/script"

// DefaultPort is the standard implicit-TLS IMAP port.
const DefaultPort = "993"

// DialTimeout bounds how long Connect waits for the initial TCP connection.
var DialTimeout = 15 * time.Second

// Client is a minimal tagged-command IMAP client.
type Client struct {
	conn net.Conn
	br   *bufio.Reader
	bw   *bufio.Writer
	tag  int
}

// Binding is an IMAPSIEVE script association.
type Binding struct {
	Mailbox string
	Script  string
}

// ResolveHostPort returns an IMAPS host:port and bare TLS server name. If
// acct.IMAPServer is set it wins; otherwise acct.Server's host is used with 993.
func ResolveHostPort(acct config.Account) (hostport, host string, err error) {
	imapServer := strings.TrimSpace(acct.IMAPServer)
	if imapServer != "" {
		if h, p, splitErr := net.SplitHostPort(imapServer); splitErr == nil {
			return net.JoinHostPort(h, p), h, nil
		}
		return net.JoinHostPort(imapServer, DefaultPort), imapServer, nil
	}

	server := strings.TrimSpace(acct.Server)
	if server == "" {
		return "", "", fmt.Errorf("account %q has no server configured", acct.Name)
	}
	if h, p, splitErr := net.SplitHostPort(server); splitErr == nil {
		_ = p
		return net.JoinHostPort(h, DefaultPort), h, nil
	}
	return net.JoinHostPort(server, DefaultPort), server, nil
}

// Connect opens an IMAPS connection, reads the greeting, and logs in.
func Connect(acct config.Account) (*Client, error) {
	if acct.Email == "" {
		return nil, fmt.Errorf("account %q has no email configured", acct.Name)
	}
	hostport, host, err := ResolveHostPort(acct)
	if err != nil {
		return nil, err
	}
	tlsConfig, err := tlsconfig.Client(host)
	if err != nil {
		return nil, fmt.Errorf("configuring IMAP TLS with %s: %w", host, err)
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: DialTimeout}, "tcp", hostport, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("connecting to IMAP %s: %w", hostport, err)
	}
	c := &Client{conn: conn, br: bufio.NewReader(conn), bw: bufio.NewWriter(conn)}
	if _, err := c.readLine(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("reading IMAP greeting from %s: %w", hostport, err)
	}
	if err := c.commandOK("LOGIN " + quote(acct.Email) + " " + quote(acct.Password)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("authenticating to IMAP as %s: %w", acct.Email, err)
	}
	return c, nil
}

// Close logs out and closes the connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	_ = c.commandOK("LOGOUT")
	return c.conn.Close()
}

// Mailboxes lists selectable and non-selectable mailbox names visible to LIST.
func (c *Client) Mailboxes() ([]string, error) {
	lines, err := c.command("LIST \"\" \"*\"")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, line := range lines {
		if name, ok := parseListMailbox(line); ok {
			out = append(out, name)
		}
	}
	return out, nil
}

// ScriptForMailbox returns the IMAPSIEVE script metadata value for mailbox. An
// empty mailbox name addresses the server-level metadata entry.
func (c *Client) ScriptForMailbox(mailbox string) (string, bool, error) {
	lines, err := c.command("GETMETADATA " + quote(mailbox) + " " + IMAPSieveScriptEntry)
	if err != nil {
		return "", false, err
	}
	for _, line := range lines {
		if value, ok := parseMetadataValue(line, IMAPSieveScriptEntry); ok {
			return value, value != "", nil
		}
	}
	return "", false, nil
}

// SetScriptForMailbox associates mailbox with script for IMAPSIEVE events. An
// empty mailbox name sets the server-level fallback script.
func (c *Client) SetScriptForMailbox(mailbox, script string) error {
	return c.commandOK("SETMETADATA " + quote(mailbox) + " (" + IMAPSieveScriptEntry + " " + quote(script) + ")")
}

// RemoveScriptForMailbox removes the IMAPSIEVE script metadata entry. An empty
// mailbox name removes the server-level fallback script.
func (c *Client) RemoveScriptForMailbox(mailbox string) error {
	return c.commandOK("SETMETADATA " + quote(mailbox) + " (" + IMAPSieveScriptEntry + " NIL)")
}

func (c *Client) commandOK(command string) error {
	_, err := c.command(command)
	return err
}

func (c *Client) command(command string) ([]string, error) {
	c.tag++
	tag := fmt.Sprintf("x%04d", c.tag)
	if _, err := fmt.Fprintf(c.bw, "%s %s\r\n", tag, command); err != nil {
		return nil, fmt.Errorf("writing command: %w", err)
	}
	if err := c.bw.Flush(); err != nil {
		return nil, fmt.Errorf("flushing command: %w", err)
	}
	var lines []string
	for {
		line, err := c.readLine()
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(line, tag+" ") {
			if strings.HasPrefix(line, tag+" OK") {
				return lines, nil
			}
			return nil, fmt.Errorf("%s", strings.TrimPrefix(line, tag+" "))
		}
		lines = append(lines, line)
	}
}

func (c *Client) readLine() (string, error) {
	line, err := c.br.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func quote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		if r == '\\' || r == '"' {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}

func parseListMailbox(line string) (string, bool) {
	if !strings.HasPrefix(line, "* LIST ") {
		return "", false
	}
	i := len(line) - 1
	for i >= 0 && line[i] == ' ' {
		i--
	}
	if i < 0 {
		return "", false
	}
	if line[i] == '"' {
		start := i
		for start > 0 {
			start--
			if line[start] == '"' && (start == 0 || line[start-1] != '\\') {
				return parseStringOrAtom(line[start : i+1])
			}
		}
		return "", false
	}
	start := i
	for start >= 0 && line[start] != ' ' {
		start--
	}
	return parseStringOrAtom(line[start+1 : i+1])
}

func parseMetadataValue(line, entry string) (string, bool) {
	if !strings.HasPrefix(line, "* METADATA ") {
		return "", false
	}
	idx := strings.Index(strings.ToLower(line), strings.ToLower(entry))
	if idx < 0 {
		return "", false
	}
	rest := strings.TrimSpace(line[idx+len(entry):])
	if strings.HasSuffix(rest, ")") {
		rest = strings.TrimSpace(strings.TrimSuffix(rest, ")"))
	}
	value, ok := parseStringOrAtom(rest)
	return value, ok
}

func parseStringOrAtom(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "NIL") {
		return "", false
	}
	if strings.HasPrefix(s, "\"") {
		v, ok := unquote(s)
		return v, ok
	}
	if i := strings.IndexAny(s, " )"); i >= 0 {
		s = s[:i]
	}
	return s, s != ""
}

func unquote(s string) (string, bool) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return "", false
	}
	var b strings.Builder
	escaped := false
	for _, r := range s[1 : len(s)-1] {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		b.WriteRune(r)
	}
	if escaped {
		return "", false
	}
	return b.String(), true
}
