// Copyright 2026 The Bazel Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"go.starlark.net/starlark"
)

// TestStringCodec exhaustively tests the encoding and decoding of
// strings by encode0 and decode0 against the properties required of
// them by RFC 8259, without regard to the particular encoding chosen.
func TestStringCodec(t *testing.T) {
	encode := func(s string) (string, error) {
		data, err := encode0(starlark.String(s))
		return string(data), err
	}
	decode := func(lit string) (string, error) {
		v, err := decode0(lit)
		if err != nil {
			return "", err
		}
		s, ok := v.(starlark.String)
		if !ok {
			return "", fmt.Errorf("got %s, want string", v.Type())
		}
		return string(s), nil
	}

	// -- encoding --

	// checkEncode checks that encode(s) is a valid UTF-8 JSON string
	// literal, and that decoding it yields s, with each byte that is
	// not part of a valid UTF-8 sequence replaced by U+FFFD.
	checkEncode := func(s string) {
		t.Helper()
		lit, err := encode(s)
		if err != nil {
			t.Errorf("encode(%q) failed: %v", s, err)
			return
		}
		if !utf8.ValidString(lit) {
			t.Errorf("encode(%q) = %q, which is not valid UTF-8", s, lit)
		}
		if !strings.HasPrefix(lit, `"`) || !json.Valid([]byte(lit)) {
			t.Errorf("encode(%q) = %q, which is not a valid JSON string literal", s, lit)
			return
		}
		want := string([]rune(s)) // replace each invalid byte by U+FFFD
		if got, err := decode(lit); err != nil {
			t.Errorf("decode(encode(%q) = %q) failed: %v", s, lit, err)
		} else if got != want {
			t.Errorf("decode(encode(%q) = %q) = %q, want %q", s, lit, got, want)
		}
	}

	// Every single byte, including the bytes 0x80-0xFF,
	// which are invalid UTF-8 in isolation.
	var allASCII, printableASCII strings.Builder
	for b := range 0x100 {
		b := byte(b)
		checkEncode(string([]byte{b}))
		if b < utf8.RuneSelf {
			allASCII.WriteByte(b)
			if 0x20 <= b && b < 0x7f {
				printableASCII.WriteByte(b)
			}
		}
	}
	checkEncode("")
	checkEncode(allASCII.String())
	checkEncode(printableASCII.String())

	// Every Unicode code point, encoded as UTF-8.
	// (Encode unpaired surrogates not as U+FFFD, but
	// as 3-byte UTF-8 sequences; this invalid code
	// is called "WTF-8".)
	var all strings.Builder
	for r := rune(0); r <= utf8.MaxRune; r++ {
		s := wtf8(r)
		checkEncode(s)
		all.WriteString(s)
	}
	checkEncode(all.String())

	// -- decoding --

	// checkDecode checks that decoding lit yields want,
	// or fails if want is the special value fails.
	const fails = "<fails>"
	checkDecode := func(lit, want string) {
		t.Helper()
		got, err := decode(lit)
		if err != nil {
			got = fails
		}
		if got != want {
			t.Errorf("decode(%q) = %q (err=%v), want %q", lit, got, err, want)
		}
	}

	checkDecode(`"abc`, fails) // unclosed

	// Every possible byte following a backslash.
	escapes := map[byte]string{
		'"':  "\"",
		'\\': "\\",
		'/':  "/",
		'b':  "\b",
		'f':  "\f",
		'n':  "\n",
		'r':  "\r",
		't':  "\t",
	}
	for b := range 0x100 {
		want, ok := escapes[byte(b)]
		if !ok {
			want = fails // includes \u without hex digits
		}
		checkDecode(`"\`+string([]byte{byte(b)})+`"`, want)
	}

	// Every byte appearing unescaped in a string literal.
	for b := range 0x100 {
		s := string([]byte{byte(b)})
		switch {
		case b == '"':
			checkDecode(`"""`, fails) // "" followed by "
		case b == '\\':
			checkDecode(`"\"`, fails) // unclosed
		case b < 0x20:
			// Unescaped control characters are not permitted
			// by RFC 8259, but are currently accepted.
			checkDecode(`"`+s+`"`, s)
		default:
			checkDecode(`"`+s+`"`, string([]rune(s)))
		}
	}

	// Every code point ≥ 0x80 appearing unescaped in a string literal.
	// Each byte of the (invalid) encoding of a surrogate
	// decodes as U+FFFD.
	for r := rune(0x80); r <= utf8.MaxRune; r++ {
		s := wtf8(r)
		checkDecode(`"`+s+`"`, string([]rune(s)))
	}

	// Every \uXXXX escape, in both cases.
	// A lone surrogate decodes as U+FFFD.
	for r := range rune(0x10000) {
		want := string(r) // (U+FFFD for a surrogate)
		checkDecode(fmt.Sprintf(`"\u%04x"`, r), want)
		checkDecode(fmt.Sprintf(`"\u%04X"`, r), want)
	}

	// Every non-hex-digit byte in each position of a \uXXXX escape.
	for b := range 0x100 {
		if strings.IndexByte("0123456789abcdefABCDEF", byte(b)) >= 0 {
			continue
		}
		for i := range 4 {
			hex := []byte("0000")
			hex[i] = byte(b)
			checkDecode(`"\u`+string(hex)+`"`, fails)
		}
	}

	// Every supplementary code point, as an escaped surrogate pair.
	for r := rune(0x10000); r <= utf8.MaxRune; r++ {
		hi, lo := utf16.EncodeRune(r)
		checkDecode(fmt.Sprintf(`"\u%04x\u%04x"`, hi, lo), string(r))
	}
}

// wtf8 returns the UTF-8 encoding of the code point r,
// generalized to surrogates, whose 3-byte encodings are invalid UTF-8.
func wtf8(r rune) string {
	if 0xD800 <= r && r < 0xE000 {
		return string([]byte{0xED, byte(0x80 | (r>>6)&0x3F), byte(0x80 | r&0x3F)})
	}
	return string(r)
}
