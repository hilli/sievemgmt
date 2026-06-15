// Copyright (C) 2020 Guido Berhoerster <guido+managesieve@berhoerster.name>
//
// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

// Package managesieve implements the MANAGESIEVE protocol as specified in
// RFC 5804.  It covers all mandatory parts of the protocol with the exception
// of the SCRAM-SHA-1 SASL mechanism.  Additional SASL authentication
// mechanisms can be provided by consumers.
package managesieve

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
)

// ParserError represents a syntax error encountered while parsing a response
// from the server.
type ParserError string

func (e ParserError) Error() string {
	return "parse error: " + string(e)
}

// ProtocolError represents a MANAGESIEVE protocol violation.
type ProtocolError string

func (e ProtocolError) Error() string {
	return "protocol error: " + string(e)
}

// NotSupportedError is returned if an operation requires an extension that is
// not available.
type NotSupportedError string

func (e NotSupportedError) Error() string {
	return "not supported: " + string(e)
}

// AuthenticationError is returned if an authentication attempt has failed.
type AuthenticationError string

func (e AuthenticationError) Error() string {
	return "authentication failed: " + string(e)
}

// A ServerError is represents an error returned by the server in the form of a
// NO response.
type ServerError struct {
	Code string // optional response code of the error
	Msg  string // optional human readable error message
}

func (e *ServerError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return "unspecified server error"
}

// The ConnClosedError is returned if the server has closed the connection.
type ConnClosedError struct {
	Code string
	Msg  string
}

func (e *ConnClosedError) Error() string {
	msg := "the server has closed to connection"
	if e.Msg != "" {
		return msg + ": " + e.Msg
	}
	return msg
}

// Tries to look up the MANAGESIEVE SRV record for the domain and returns an
// slice of strings containing hostnames and ports. If no SRV record was found
// it falls back to the given domain name and port 4190.
func LookupService(domain string) ([]string, error) {
	_, addrs, err := net.LookupSRV("sieve", "tcp", domain)
	if err != nil {
		if dnserr, ok := err.(*net.DNSError); ok {
			if dnserr.IsNotFound {
				// no SRV record found, fall back to port 4190
				services := [1]string{domain + ":4190"}
				return services[:], nil
			}
		}
		return nil, err
	}
	services := make([]string, 0, len(addrs))
	// addrs is already ordered by priority
	for _, addr := range addrs {
		services = append(services,
			fmt.Sprintf("%s:%d", addr.Target, addr.Port))
	}
	return services, nil
}

// Checks whtether the given string conforms to the "Unicode Format for Network
// Interchange" specified in RFC 5198.
func IsNetUnicode(s string) bool {
	for _, c := range s {
		if c <= 0x1f || (c >= 0x7f && c <= 0x9f) ||
			c == 0x2028 || c == 0x2029 {
			return false
		}
	}
	return true
}

func quoteString(s string) string {
	return fmt.Sprintf("{%d+}\r\n%s", len(s), s)
}

// Client represents a client connection to a MANAGESIEVE server.
type Client struct {
	conn       net.Conn
	p          *parser
	isAuth     bool
	capa       map[string]string
	serverName string
}

// Dial creates a new connection to a MANAGESIEVE server. The given addr must
// contain both a hostname or IP address and post.
func Dial(addr string) (*Client, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewClient(conn, host)
}

// NewClient create a new client based on an existing connection to a
// MANAGESIEVE server where host specifies the hostname of the remote end of
// the connection.
func NewClient(conn net.Conn, host string) (*Client, error) {
	s := newScanner(conn)
	p := &parser{s}
	c := &Client{
		conn:       conn,
		p:          p,
		serverName: host,
	}

	r, err := c.p.readReply()
	if err != nil {
		c.Close()
		return nil, err
	}
	switch r.resp {
	case responseOk:
		c.capa, err = parseCapabilities(r)
	case responseNo:
		return c, &ServerError{r.code, r.msg}
	case responseBye:
		return c, &ConnClosedError{r.code, r.msg}
	}
	return c, err
}

// Implementation returns the name and version of the implementation as
// reported by the server.
func (c *Client) Implementation() string {
	return c.capa["IMPLEMENTATION"]
}

// SupportsRFC5804 returns true if the server conforms to RFC 5804.
func (c *Client) SupportsRFC5804() bool {
	_, ok := c.capa["VERSION"]
	return ok
}

// SupportsTLS returns true if the server supports TLS connections via the
// STARTTLS command.
func (c *Client) SupportsTLS() bool {
	_, ok := c.capa["STARTTLS"]
	return ok
}

// Extensions returns the Sieve script extensions supported by the Sieve engine.
func (c *Client) Extensions() []string {
	return strings.Fields(c.capa["SIEVE"])
}

// MaxRedirects returns the limit on the number of Sieve "redirect" during a
// single evaluation.
func (c *Client) MaxRedirects() int {
	n, err := strconv.ParseUint(c.capa["MAXREDIRECTS"], 10, 32)
	if err != nil {
		return 0
	}
	return int(n)
}

// NotifyMethods returns the URI schema parts for supported notification
// methods.
func (c *Client) NotifyMethods() []string {
	return strings.Fields(c.capa["NOTIFY"])
}

// SASLMechanisms returns the SASL authentication mechanism supported by the
// server. This may change depending on whether a TLS connection is used.
func (c *Client) SASLMechanisms() []string {
	splitFunc := func(r rune) bool {
		return r == ' '
	}
	return strings.FieldsFunc(c.capa["SASL"], splitFunc)
}

func (c *Client) cmd(args ...interface{}) (*reply, error) {
	// write each arg separated by a space and terminated by CR+LF
	for i, arg := range args {
		if i > 0 {
			if _, err := c.conn.Write([]byte{' '}); err != nil {
				return nil, err
			}
		}
		if _, err := fmt.Fprint(c.conn, arg); err != nil {
			return nil, err
		}
	}
	if _, err := c.conn.Write([]byte("\r\n")); err != nil {
		return nil, err
	}

	r, err := c.p.readReply()
	if err != nil {
		return nil, err
	}
	if r.resp == responseNo {
		return r, &ServerError{r.code, r.msg}
	} else if r.resp == responseBye {
		return r, &ConnClosedError{r.code, r.msg}
	}
	return r, nil
}

// StartTLS upgrades the connection to use TLS encryption based on the given
// configuration.
func (c *Client) StartTLS(config *tls.Config) error {
	if _, ok := c.conn.(*tls.Conn); ok {
		return ProtocolError("already using a TLS connection")
	}
	if c.isAuth {
		return ProtocolError("cannot STARTTLS in authenticated state")
	}
	if _, ok := c.capa["STARTTLS"]; !ok {
		return NotSupportedError("STARTTLS")
	}
	if _, err := c.cmd("STARTTLS"); err != nil {
		return err
	}
	c.conn = tls.Client(c.conn, config)
	s := newScanner(c.conn)
	c.p = &parser{s}

	r, err := c.p.readReply()
	if err != nil {
		return err
	}
	// capabilities are no longer valid if STARTTLS succeeded
	c.capa, err = parseCapabilities(r)
	return err
}

// TLSConnectionState returns the ConnectionState of the current TLS
// connection.
func (c *Client) TLSConnectionState() (state tls.ConnectionState, ok bool) {
	tc, ok := c.conn.(*tls.Conn)
	if !ok {
		return
	}
	return tc.ConnectionState(), ok
}

// Authenticate authenticates a client using the given authentication
// mechanism. In case of an AuthenticationError the client remains in a defined
// state and can continue to be used.
func (c *Client) Authenticate(a Auth) error {
	encoding := base64.StdEncoding
	_, isTLS := c.conn.(*tls.Conn)
	info := &ServerInfo{c.serverName, isTLS, c.SASLMechanisms()}
	mech, resp, err := a.Start(info)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(c.conn, "AUTHENTICATE \"%s\" \"%s\"\r\n",
		mech, encoding.EncodeToString(resp)); err != nil {
		return err
	}

	var line []*token
	// handle SASL challenge-response messages exchanged as base64-encoded
	// strings until the server sends a MANAGESIEVE response which may
	// contain some final SASL data
	for {
		line, err = c.p.readLine()
		if err != nil {
			return err
		}

		if c.p.isResponseLine(line) {
			break
		} else if len(line) != 1 ||
			(line[0].typ != tokenQuotedString &&
				line[0].typ != tokenLiteralString) {
			// Some servers (e.g. mox) emit an unsolicited capabilities
			// list after a successful AUTHENTICATE, before the final
			// response line (RFC 5804, Section 1.7). Such lines are not
			// single SASL data strings, so skip them; the up-to-date
			// capabilities are fetched again below once authenticated.
			continue
		}
		msg, err := encoding.DecodeString(line[0].literal)
		if err != nil {
			return ParserError("failed to decode SASL data: " +
				err.Error())
		}

		// perform next step in authentication process
		resp, authErr := a.Next(msg, true)
		if authErr != nil {
			// this error should be recoverable, abort
			// authentication
			if _, err := fmt.Fprintf(c.conn, "\"*\"\r\n"); err != nil {
				return err
			}

			line, err = c.p.readLine()
			if err != nil {
				return err
			}
			if r, err := c.p.parseResponseLine(line); err != nil {
				return err
			} else if r.resp != responseNo {
				return ProtocolError("invalid response to aborted authentication: expected NO")
			}

			return AuthenticationError(authErr.Error())
		}

		// send SASL response
		if _, err := fmt.Fprintf(c.conn, "\"%s\"\r\n",
			encoding.EncodeToString(resp)); err != nil {
			return err
		}
	}

	// handle MANAGESIEVE response
	r, err := c.p.parseResponseLine(line)
	if err != nil {
		return err
	}
	if r.resp == responseNo {
		return AuthenticationError(r.msg)
	} else if r.resp == responseBye {
		return &ConnClosedError{r.code, r.msg}
	}

	// check for SASL response code with final SASL data as the response
	// code argument
	if r.code == "SASL" {
		if len(r.codeArgs) != 1 {
			return ParserError("failed to parse SASL code argument: expected a single argument")
		}
		msg64 := r.codeArgs[0]
		msg, err := encoding.DecodeString(msg64)
		if err != nil {
			return ParserError("failed to decode SASL code argument: " + err.Error())
		}

		if _, err = a.Next(msg, false); err != nil {
			return AuthenticationError(err.Error())
		}
	}

	// capabilities are no longer valid after succesful authentication
	r, err = c.cmd("CAPABILITY")
	if err != nil {
		return err
	}
	c.capa, err = parseCapabilities(r)
	return err
}

// HaveSpace queries the server if there is sufficient space to store a script
// with the given name and size.  An already existing script with the same name
// will be treated as if it were replaced with a script of the given size.
func (c *Client) HaveSpace(name string, size int64) (bool, error) {
	if size < 0 {
		return false,
			ProtocolError(fmt.Sprintf("invalid script size: %d",
				size))
	}
	if size > math.MaxInt32 {
		return false, ProtocolError("script exceeds maximum size")
	}
	r, err := c.cmd("HAVESPACE", quoteString(name), size)
	if err != nil {
		if r.code == "QUOTA" || r.code == "QUOTA/MAXSIZE" {
			err = nil
		}
	}
	return r.resp == responseOk, err
}

// PutScript stores the script content with the given name on the server.  An
// already existing script with the same name will be replaced.
func (c *Client) PutScript(name, content string) (warnings string, err error) {
	if !IsNetUnicode(name) {
		err = ProtocolError("script name must comply with Net-Unicode")
		return
	}
	r, err := c.cmd("PUTSCRIPT", quoteString(name), quoteString(content))
	if err != nil {
		return
	}
	if r.code == "WARNINGS" {
		warnings = r.msg
	}
	return
}

// ListScripts returns the names of all scripts on the server and the name of
// the currently active script.  If there is no active script it returns the
// empty string.
func (c *Client) ListScripts() ([]string, string, error) {
	r, err := c.cmd("LISTSCRIPTS")
	if err != nil {
		return nil, "", err
	}

	var scripts []string = make([]string, 0)
	var active string
	for _, tokens := range r.lines {
		if tokens[0].typ != tokenQuotedString &&
			tokens[0].typ != tokenLiteralString {
			return nil, "", ParserError("failed to parse script list: expected string")
		}
		switch len(tokens) {
		case 2:
			if tokens[1].typ != tokenAtom ||
				tokens[1].literal != "ACTIVE" {
				return nil, "", ParserError("failed to parse script list: expected atom ACTIVE")
			}
			active = tokens[0].literal
			fallthrough
		case 1:
			scripts = append(scripts, tokens[0].literal)
		default:
			return nil, "", ParserError("failed to parse script list: trailing data")
		}
	}
	return scripts, active, nil
}

// ActivateScript activates a script. Only one script can be active at the same
// time, activating a script will deactivate the previously active script. If
// the name is the empty string the currently active script will be
// deactivated.
func (c *Client) ActivateScript(name string) error {
	_, err := c.cmd("SETACTIVE", quoteString(name))
	return err
}

// GetScript returns the content of the script with the given name.
func (c *Client) GetScript(name string) (string, error) {
	r, err := c.cmd("GETSCRIPT", quoteString(name))
	if err != nil {
		return "", err
	}
	if len(r.lines) != 1 ||
		(r.lines[0][0].typ != tokenQuotedString &&
			r.lines[0][0].typ != tokenLiteralString) {
		return "", ParserError("failed to parse script: expected string")
	}
	return r.lines[0][0].literal, nil
}

// DeleteScript deletes the script with the given name from the server.
func (c *Client) DeleteScript(name string) error {
	_, err := c.cmd("DELETESCRIPT", quoteString(name))
	return err
}

// RenameScript renames a script on the server. This operation is only
// available if the server conforms to RFC 5804.
func (c *Client) RenameScript(oldName, newName string) error {
	if !c.SupportsRFC5804() {
		return NotSupportedError("RENAMESCRIPT")
	}
	if !IsNetUnicode(newName) {
		return ProtocolError("script name must comply with Net-Unicode")
	}
	_, err := c.cmd("RENAMESCRIPT", quoteString(oldName),
		quoteString(newName))
	return err
}

// CheckScript checks if the given script contains any errors. This operation
// is only available if the server conforms to RFC 5804.
func (c *Client) CheckScript(content string) (warnings string, err error) {
	if !c.SupportsRFC5804() {
		err = NotSupportedError("CHECKSCRIPT")
		return
	}
	r, err := c.cmd("CHECKSCRIPT", quoteString(content))
	if err != nil {
		return
	}
	if r.code == "WARNINGS" {
		warnings = r.msg
	}
	return
}

// Noop does nothing but contact the server and can be used to prevent timeouts
// and to check whether the connection is still alive. This operation is only
// available if the server conforms to RFC 5804.
func (c *Client) Noop() error {
	if !c.SupportsRFC5804() {
		return NotSupportedError("NOOP")
	}
	_, err := c.cmd("NOOP")
	return err
}

// Close closes the connection to the server immediately without informing the
// remote end that the client has finished.  Under normal circumstances Logout
// should be used instead.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Logout first indicates to the server that the client is finished and
// subsequently closes the connection. No further commands can be sent after
// this.
func (c *Client) Logout() error {
	_, err := c.cmd("LOGOUT")
	cerr := c.Close()
	if err == nil {
		err = cerr
	}
	return err
}
