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
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const ReadLimit = 1 * 1024 * 1024 // 1 MiB

type tokenType int

const (
	tokenInvalid tokenType = iota
	tokenCRLF
	tokenLeftParenthesis
	tokenRightParenthesis
	tokenAtom
	tokenQuotedString
	tokenLiteralString
)

type token struct {
	typ     tokenType
	literal string
}

func (t token) String() string {
	switch t.typ {
	case tokenInvalid:
		return "Invalid"
	case tokenLeftParenthesis:
		return "LeftParenthesis: " + t.literal
	case tokenRightParenthesis:
		return "RightParenthesis: " + t.literal
	case tokenAtom:
		return "Atom: " + t.literal
	case tokenCRLF:
		return fmt.Sprintf("CRLF: %q", t.literal)
	case tokenQuotedString:
		return fmt.Sprintf("QuotedString: %q", t.literal)
	case tokenLiteralString:
		return fmt.Sprintf("LiteralString: %q", t.literal)
	}
	return fmt.Sprintf("unknown token: %q", t.literal)
}

type scanner struct {
	lr *io.LimitedReader // do not read from this, only for access to N
	br *bufio.Reader     // wraps LimitReader
}

func newScanner(r io.Reader) *scanner {
	lr := &io.LimitedReader{R: r, N: ReadLimit}
	br := bufio.NewReader(lr)
	return &scanner{lr, br}
}

func (s *scanner) scanCRLF() (*token, error) {
	c, _, err := s.br.ReadRune()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	literal := string(c)
	// accept LF without CR
	if c == '\r' {
		c, _, err = s.br.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		literal += string(c)
	}
	if c != '\n' {
		return nil, ParserError(fmt.Sprintf(`expected '\n', got %q`, c))
	}
	return &token{typ: tokenCRLF, literal: literal}, nil
}

func (s *scanner) scanParenthesis() (*token, error) {
	c, _, err := s.br.ReadRune()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	var typ tokenType
	if c == '(' {
		typ = tokenLeftParenthesis
	} else if c == ')' {
		typ = tokenRightParenthesis
	} else {
		return nil,
			ParserError(fmt.Sprintf("expected parenthesis, got %q",
				c))
	}
	return &token{typ: typ, literal: string(c)}, nil
}

func isAtomRune(c rune) bool {
	return c == '!' ||
		(c >= 0x23 && c <= 0x27) ||
		(c >= 0x2a && c <= 0x5b) ||
		(c >= 0x5d && c <= 0x7a) ||
		(c >= 0x7c && c <= 0x7e)
}

func (s *scanner) scanAtom() (*token, error) {
	var sb strings.Builder
	var c rune
	for {
		c, _, err := s.br.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		if isAtomRune(c) {
			sb.WriteRune(unicode.ToUpper(c))
		} else {
			s.br.UnreadRune()
			break
		}
	}
	if sb.Len() == 0 {
		return nil, ParserError(fmt.Sprintf("expected atom, got %q", c))
	}
	return &token{typ: tokenAtom, literal: sb.String()}, nil
}

func (s *scanner) scanQuotedString() (*token, error) {
	c, _, err := s.br.ReadRune()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if c != '"' {
		return nil, ParserError(fmt.Sprintf("expected '\"', got %q", c))
	}
	// Per RFC 5804 a quoted string may contain backslash-escaped
	// characters: '\"' for a literal double quote and '\\' for a literal
	// backslash. Decode these escapes rather than terminating the string
	// at the first quote, which would mis-parse escaped quotes in
	// server messages.
	var sb strings.Builder
	for {
		r, _, err := s.br.ReadRune()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		switch r {
		case '"':
			return &token{typ: tokenQuotedString, literal: sb.String()}, nil
		case '\\':
			next, _, err := s.br.ReadRune()
			if err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return nil, err
			}
			sb.WriteRune(next)
		default:
			sb.WriteRune(r)
		}
	}
}

func (s *scanner) scanLiteralString() (*token, error) {
	c, _, err := s.br.ReadRune()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if c != '{' {
		return nil, ParserError(fmt.Sprintf("expected '{', got %q", c))
	}
	nstr, err := s.br.ReadString('}')
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	n, err := strconv.ParseUint(nstr[:len(nstr)-1], 10, 32)
	if err != nil {
		return nil, ParserError("failed to parse literal string length: " + err.Error())
	}
	if n > uint64(s.lr.N) {
		return nil, ParserError(fmt.Sprintf("string too long: %d", n))
	}

	if _, err := s.scanCRLF(); err != nil {
		return nil, err
	}

	b := make([]byte, n)
	_, err = io.ReadFull(s.br, b)
	ls := string(b)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}

	return &token{typ: tokenLiteralString, literal: ls}, nil
}

func (s *scanner) skipSpace() error {
	for {
		b, err := s.br.ReadByte()
		if err != nil {
			if err == io.EOF {
				err = io.ErrUnexpectedEOF
			}
			return err
		}

		if b != ' ' {
			s.br.UnreadByte()
			break
		}
	}

	return nil
}

func (s *scanner) scan() (*token, error) {
	if err := s.skipSpace(); err != nil {
		return nil, err
	}

	buf, err := s.br.Peek(1)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	b := buf[0]
	switch {
	case b == '\r':
		fallthrough
	case b == '\n':
		return s.scanCRLF()
	case b == '"':
		return s.scanQuotedString()
	case b == '{':
		return s.scanLiteralString()
	case b == '(':
		fallthrough
	case b == ')':
		return s.scanParenthesis()
	case isAtomRune(rune(b)):
		return s.scanAtom()
	}
	return nil, ParserError(fmt.Sprintf("invalid character: %q", b))
}
