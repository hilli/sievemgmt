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
	"strings"
)

type response int

const (
	responseInvalid response = iota
	responseOk
	responseNo
	responseBye
)

func lookupResponse(s string) response {
	switch s {
	case "OK":
		return responseOk
	case "NO":
		return responseNo
	case "BYE":
		return responseBye
	}
	return responseInvalid
}

type reply struct {
	lines    [][]*token
	resp     response
	code     string
	codeArgs []string
	msg      string
}

func parseCapabilities(r *reply) (map[string]string, error) {
	capa := make(map[string]string)
	for _, tokens := range r.lines {
		var k, v string
		if tokens[0].typ != tokenQuotedString &&
			tokens[0].typ != tokenLiteralString {
			return nil, ParserError("failed to parse capability name: expected string")
		}
		k = strings.ToUpper(tokens[0].literal)

		if len(tokens) > 1 {
			if tokens[1].typ != tokenQuotedString &&
				tokens[1].typ != tokenLiteralString {
				return nil, ParserError("failed to parse capability value: expected string")
			}
			v = tokens[1].literal
		}
		capa[k] = v
	}
	return capa, nil
}

type parser struct {
	s *scanner
}

func (p *parser) isResponseLine(tokens []*token) bool {
	return tokens[0].typ == tokenAtom &&
		lookupResponse(tokens[0].literal) != responseInvalid
}

func (p *parser) parseResponseLine(tokens []*token) (*reply, error) {
	var i int
	next := func() (*token, bool) {
		if i >= len(tokens) {
			return nil, false
		}
		tok := tokens[i]
		i++
		return tok, true
	}

	// response
	tok, cont := next()
	r := &reply{resp: lookupResponse(tok.literal)}

	// code starts with left parenthesis
	tok, cont = next()
	if !cont {
		// only response without code and/or message
		return r, nil
	}
	if tok.typ == tokenLeftParenthesis {
		// code atom
		tok, cont = next()
		if !cont || tok.typ != tokenAtom {
			return nil, ParserError("failed to parse response code: expected atom")
		}
		r.code = tok.literal

		// followed by zero or more string arguments
		for {
			tok, cont = next()
			if !cont {
				return nil, ParserError("failed to parse response code: unexpected end of line")
			}
			if tok.typ != tokenQuotedString &&
				tok.typ != tokenLiteralString {
				break
			}
			r.codeArgs = append(r.codeArgs, tok.literal)
		}

		// terminated by a right parenthesis
		if tok.typ != tokenRightParenthesis {
			return nil, ParserError("failed to parse response code: expected right parenthesis")
		}

		tok, cont = next()
		if !cont {
			// response with code but no message
			return r, nil
		}
	}

	// message string
	if tok.typ != tokenQuotedString &&
		tok.typ != tokenLiteralString {
		return nil, ParserError("failed to parse response message: expected string")
	}
	r.msg = strings.TrimSpace(tok.literal)

	// end of line
	if _, cont = next(); cont {
		return nil, ParserError("failed to parse response line: unexpected trailing data")
	}

	return r, nil
}

func (p *parser) readLine() ([]*token, error) {
	tokens := make([]*token, 0)
	for {
		tok, err := p.s.scan()
		if err != nil {
			return nil, err
		}
		if tok.typ == tokenCRLF {
			break
		}
		tokens = append(tokens, tok)
	}

	return tokens, nil
}

func (p *parser) readReply() (*reply, error) {
	var r *reply
	var lines [][]*token = make([][]*token, 0, 1)
	for {
		tokens, err := p.readLine()
		if err != nil {
			return nil, err
		}
		if len(tokens) == 0 {
			return nil, ParserError("unexpected empty line")
		}
		// check for response tokens
		if p.isResponseLine(tokens) {
			r, err = p.parseResponseLine(tokens)
			if err != nil {
				return nil, err
			}
			r.lines = lines
			break
		}
		lines = append(lines, tokens)
	}

	return r, nil
}
