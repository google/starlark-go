// Copyright 2017 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax

import (
	"strings"
	"testing"
)

var quoteTests = []struct {
	q   string // quoted
	s   string // unquoted (actual string)
	std bool   // q is standard form for s
}{
	{`""`, "", true},
	{`''`, "", false},
	{`"hello"`, `hello`, true},
	{`'hello'`, `hello`, false},
	{`"quote\"here"`, `quote"here`, true},
	{`'quote"here'`, `quote"here`, false},
	{`"quote'here"`, `quote'here`, true},
	{`'quote\'here'`, `quote'here`, false},
	{`"""hello " ' world "" asdf ''' foo"""`, `hello " ' world "" asdf ''' foo`, true},
	{`"""hello
world"""`, "hello\nworld", true},

	{`"\a\b\f\n\r\t\v\000\377"`, "\a\b\f\n\r\t\v\000\xFF", true},
	{`"\a\b\f\n\r\t\v\x00\xff"`, "\a\b\f\n\r\t\v\000\xFF", false},
	{`"\a\b\f\n\r\t\v\000\xFF"`, "\a\b\f\n\r\t\v\000\xFF", false},
	{`"\a\b\f\n\r\t\v\000\377\"'\\\003\200"`, "\a\b\f\n\r\t\v\x00\xFF\"'\\\x03\x80", true},
	{`"\a\b\f\n\r\t\v\x00\xff\"'\\\x03\x80"`, "\a\b\f\n\r\t\v\x00\xFF\"'\\\x03\x80", false},
	{`"\a\b\f\n\r\t\v\000\xFF\"'\\\x03\x80"`, "\a\b\f\n\r\t\v\x00\xFF\"'\\\x03\x80", false},
	{`"\a\b\f\n\r\t\v\000\xFF\"\\\x03\x80"`, "\a\b\f\n\r\t\v\x00\xFF\"\\\x03\x80", false},
	{
		`"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    \x27\\1\x27,/g' >> $@; "`,
		"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    '\\1',/g' >> $@; ",
		false,
	},
	{
		`"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    '\\1',/g' >> $@; "`,
		"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    '\\1',/g' >> $@; ",
		true,
	},
}

func TestQuote(t *testing.T) {
	for _, tt := range quoteTests {
		if !tt.std {
			continue
		}
		q := quote(tt.s, strings.HasPrefix(tt.q, `"""`))
		if q != tt.q {
			t.Errorf("quote(%#q) = %s, want %s", tt.s, q, tt.q)
		}
	}
}

func TestUnquote(t *testing.T) {
	for _, tt := range quoteTests {
		s, triple, err := unquote(tt.q)
		wantTriple := strings.HasPrefix(tt.q, `"""`) || strings.HasPrefix(tt.q, `'''`)
		if s != tt.s || triple != wantTriple || err != nil {
			t.Errorf("unquote(%s) = %#q, %v, %v want %#q, %v, nil", tt.q, s, triple, err, tt.s, wantTriple)
		}
	}
}
