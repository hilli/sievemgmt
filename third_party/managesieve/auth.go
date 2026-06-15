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

package managesieve

import (
	"errors"
	"fmt"
)

// this API is inspired by the SASL authentication API in net/smtp

// ServerInfo stores information about the ManageSieve server.
type ServerInfo struct {
	Name string   // hostname of the server
	TLS  bool     // whether a verified TLS connection is used
	Auth []string // authentication methods advertised in capabilities
}

// Check whether the server supports the wanted SASL authentication mechanism.
func (s *ServerInfo) HaveAuth(wanted string) bool {
	for _, m := range s.Auth {
		if m == wanted {
			return true
		}
	}
	return false
}

type Auth interface {
	// Initiate SASL authentication.  A non-nil response will be sent in
	// response to an empty challenge from the server if mandated by the
	// authentication mechanism.  The name of the SASL authentication
	// mechanism is returned in mechanism.  If an error is returned SASL
	// authentication will be aborted and an AuthenticationError will be
	// returned to the caller.
	Start(server *ServerInfo) (mechanism string, response []byte, err error)
	// Handle a challenge received from the server, if more is true the
	// server expects a response, otherwise the response should be nil. If
	// an error is returned SASL authentication will be aborted and an
	// AuthenticationError will be returned to the caller.
	Next(challenge []byte, more bool) (response []byte, err error)
}

var (
	// ErrPlainAuthNotSupported is returned if the server does not support
	// the SASL PLAIN authentication mechanism.
	ErrPlainAuthNotSupported = errors.New("the server does not support PLAIN authentication")
	// ErrPlainAuthTLSRequired is returned when the SASL PLAIN
	// authentication mechanism is used without TLS against a server other
	// than localhost.
	ErrPlainAuthTLSRequired = errors.New("PLAIN authentication requires a TLS connection")
)

// HostNameVerificationError is returned when the hostname which was passed to
// the Auth implementation could not be verified against the TLS certificate.
type HostNameVerificationError struct {
	ExpectedHost, ActualHost string
}

func (e *HostNameVerificationError) Error() string {
	return fmt.Sprintf("host name mismatch: %s != %s", e.ActualHost,
		e.ExpectedHost)
}

type plainAuth struct {
	identity string
	username string
	password string
	host     string
}

func (a *plainAuth) Start(server *ServerInfo) (string, []byte, error) {
	if !server.HaveAuth("PLAIN") {
		return "PLAIN", nil, ErrPlainAuthNotSupported
	}

	// enforce TLS for non-local servers in order to avoid leaking
	// credentials via unencrypted connections or DNS spoofing
	if !server.TLS && server.Name != "localhost" &&
		server.Name != "127.0.0.1" && server.Name != "::1" {
		return "PLAIN", nil, ErrPlainAuthTLSRequired
	}

	// verify server hostname before sending credentials
	if server.Name != a.host {
		return "PLAIN", nil,
			&HostNameVerificationError{a.host, server.Name}
	}

	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

func (a *plainAuth) Next(challenge []byte, more bool) ([]byte, error) {
	return nil, nil
}

// PlainAuth provides an Auth implementation of SASL PLAIN authentication as
// specified in RFC 4616 using the provided authorization identity, username
// and password. If the identity is an empty string the server will derive an
// identity from the credentials.
func PlainAuth(identity, username, password, host string) Auth {
	return &plainAuth{identity, username, password, host}
}
