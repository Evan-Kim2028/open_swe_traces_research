// Copyright 2013-2024 The Msgkit Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Customized heavily from
// https://github.com/BurntSushi/toml/blob/master/lex.go, which is based on
// Rob Pike's talk: http://cuddle.googlecode.com/hg/talk/lex.html

// The format supported is less restrictive than today's formats.
// Supports mixed Arrays [], nested Maps {}, multiple comment types (# and //)
// Also supports key value assignments using '=' or ':' or whiteSpace()
//   e.g. foo = 2, foo : 2, foo 2
// maps can be assigned with no key separator as well
// semicolons as value terminators in key/value assignments are optional
//
// see lex_test.go for more examples.

package conf

import (
	_ "encoding/hex"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type itemType int

const (
	itemError itemType = iota
	itemNIL            // used in the parser to indicate no type
	itemEOF
	itemKey
	itemText
	itemString
	itemBool
	itemInteger
	itemFloat
	itemDatetime
	itemArrayStart
	itemArrayEnd
	itemMapStart
	itemMapEnd
	itemCommentStart
	itemVariable
	itemInclude
)

const (
	eof               = 0
	mapStart          = '{'
	mapEnd            = '}'
	keySepEqual       = '='
	keySepColon       = ':'
	arrayStart        = '['
	arrayEnd          = ']'
	arrayValTerm      = ','
	mapValTerm        = ','
	commentHashStart  = '#'
	commentSlashStart = '/'
	dqStringStart     = '"'
	dqStringEnd       = '"'
	sqStringStart     = '\''
	sqStringEnd       = '\''
	optValTerm        = ';'
	topOptStart       = '{'
	topOptValTerm     = ','
	topOptTerm        = '}'
	blockStart        = '('
	blockEnd          = ')'
	mapEndString      = string(mapEnd)
)

type stateFn func(lx *lexer) stateFn

type lexer struct {
	input string
	start int
	pos   int
	width int
	line  int
	state stateFn
	items chan item

	// A stack of state functions used to maintain context.
	// The idea is to reuse parts of the state machine in various places.
	// For example, values can appear at the top level or within arbitrarily
	// nested arrays. The last state on the stack is used after a value has
	// been lexed. Similarly for comments.
	stack []stateFn

	// Used for processing escapable substrings in double-quoted and raw strings
	stringParts   []string
	stringStateFn stateFn

	// lstart is the start position of the current line.
	lstart int

	// ilstart is the start position of the line from the current item.
	ilstart int
}

type item struct {
	typ  itemType
	val  string
	line int
	pos  int
}

func (lx *lexer) nextItem() item {
	for {
		select {
		case item := <-lx.items:
			return item
		default:
			lx.state = lx.state(lx)
		}
	}
}

func lex(input string) *lexer {
	lx := &lexer{
		input:       input,
		state:       lexTop,
		line:        1,
		items:       make(chan item, 10),
		stack:       make([]stateFn, 0, 10),
		stringParts: []string{},
	}
	return lx
}

func (lx *lexer) push(state stateFn) {
	lx.stack = append(lx.stack, state)
}

func (lx *lexer) pop() stateFn {
	if len(lx.stack) == 0 {
		return lx.errorf("BUG in lexer: no states to pop.")
	}
	li := len(lx.stack) - 1
	last := lx.stack[li]
	lx.stack = lx.stack[0:li]
	return last
}

func (lx *lexer) emit(typ itemType) {
	val := strings.Join(lx.stringParts, "") + lx.input[lx.start:lx.pos]
	// Position of item in line where it started.
	pos := lx.pos - lx.ilstart - len(val)
	lx.items <- item{typ, val, lx.line, pos}
	lx.start = lx.pos
	lx.ilstart = lx.lstart
}

func (lx *lexer) emitString() { panic("excised: emitString") }

func (lx *lexer) addCurrentStringPart(offset int) { panic("excised: addCurrentStringPart") }

func (lx *lexer) addStringPart(s string) stateFn { panic("excised: addStringPart") }

func (lx *lexer) hasEscapedParts() bool { panic("excised: hasEscapedParts") }

func (lx *lexer) next() (r rune) {
	if lx.pos >= len(lx.input) {
		lx.width = 0
		return eof
	}

	if lx.input[lx.pos] == '\n' {
		lx.line++

		// Mark start position of current line.
		lx.lstart = lx.pos
	}
	r, lx.width = utf8.DecodeRuneInString(lx.input[lx.pos:])
	lx.pos += lx.width

	return r
}

// ignore skips over the pending input before this point.
func (lx *lexer) ignore() {
	lx.start = lx.pos
	lx.ilstart = lx.lstart
}

// backup steps back one rune. Can be called only once per call of next.
func (lx *lexer) backup() {
	lx.pos -= lx.width
	if lx.pos < len(lx.input) && lx.input[lx.pos] == '\n' {
		lx.line--
	}
}

// peek returns but does not consume the next rune in the input.
func (lx *lexer) peek() rune {
	r := lx.next()
	lx.backup()
	return r
}

// errorf stops all lexing by emitting an error and returning `nil`.
// Note that any value that is a character is escaped if it's a special
// character (new lines, tabs, etc.).
func (lx *lexer) errorf(format string, values ...any) stateFn {
	for i, value := range values {
		if v, ok := value.(rune); ok {
			values[i] = escapeSpecial(v)
		}
	}

	// Position of error in current line.
	pos := lx.pos - lx.lstart
	lx.items <- item{
		itemError,
		fmt.Sprintf(format, values...),
		lx.line,
		pos,
	}
	return nil
}

// lexTop consumes elements at the top level of data structure.
func lexTop(lx *lexer) stateFn {
	r := lx.next()
	if unicode.IsSpace(r) {
		return lexSkip(lx, lexTop)
	}

	switch r {
	case topOptStart:
		lx.push(lexTop)
		return lexSkip(lx, lexBlockStart)
	case commentHashStart:
		lx.push(lexTop)
		return lexCommentStart
	case commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexTop)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case eof:
		if lx.pos > lx.start {
			return lx.errorf("Unexpected EOF.")
		}
		lx.emit(itemEOF)
		return nil
	}

	// At this point, the only valid item can be a key, so we back up
	// and let the key lexer do the rest.
	lx.backup()
	lx.push(lexTopValueEnd)
	return lexKeyStart
}

// lexTopValueEnd is entered whenever a top-level value has been consumed.
// It must see only whitespace, and will turn back to lexTop upon a new line.
// If it sees EOF, it will quit the lexer successfully.
func lexTopValueEnd(lx *lexer) stateFn { panic("excised: lexTopValueEnd") }

func lexBlockStart(lx *lexer) stateFn {
	r := lx.next()
	if unicode.IsSpace(r) {
		return lexSkip(lx, lexBlockStart)
	}

	switch r {
	case topOptStart:
		lx.push(lexBlockEnd)
		return lexSkip(lx, lexBlockStart)
	case topOptTerm:
		lx.ignore()
		return lx.pop()
	case commentHashStart:
		lx.push(lexBlockStart)
		return lexCommentStart
	case commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexBlockStart)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case eof:
		if lx.pos > lx.start {
			return lx.errorf("Unexpected EOF.")
		}
		lx.emit(itemEOF)
		return nil
	}

	// At this point, the only valid item can be a key, so we back up
	// and let the key lexer do the rest.
	lx.backup()
	lx.push(lexBlockValueEnd)
	return lexKeyStart
}

// lexBlockValueEnd is entered whenever a block-level value has been consumed.
// It must see only whitespace, and will turn back to lexBlockStart upon a new line.
// If it sees EOF, it will quit the lexer successfully.
func lexBlockValueEnd(lx *lexer) stateFn { panic("excised: lexBlockValueEnd") }

// lexBlockEnd is entered whenever a block-level value has been consumed.
// It must see only whitespace, and will turn back to lexTop upon a "}".
func lexBlockEnd(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == commentHashStart:
		// a comment will read to a new line for us.
		lx.push(lexBlockStart)
		return lexCommentStart
	case r == commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexBlockStart)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case isNL(r) || isWhitespace(r):
		return lexBlockEnd
	case r == optValTerm || r == topOptValTerm:
		lx.ignore()
		return lexBlockStart
	case r == topOptTerm:
		lx.ignore()
		return lx.pop()
	}
	return lx.errorf("Expected a block-level to end with a '}', but got '%v' instead.", r)
}

// lexKeyStart consumes a key name up until the first non-whitespace character.
// lexKeyStart will ignore whitespace. It will also eat enclosing quotes.
func lexKeyStart(lx *lexer) stateFn {
	r := lx.peek()
	switch {
	case isKeySeparator(r):
		return lx.errorf("Unexpected key separator '%v'", r)
	case unicode.IsSpace(r):
		lx.next()
		return lexSkip(lx, lexKeyStart)
	case r == dqStringStart:
		lx.next()
		return lexSkip(lx, lexDubQuotedKey)
	case r == sqStringStart:
		lx.next()
		return lexSkip(lx, lexQuotedKey)
	}
	lx.ignore()
	lx.next()
	return lexKey
}

// lexDubQuotedKey consumes the text of a key between quotes.
func lexDubQuotedKey(lx *lexer) stateFn {
	r := lx.peek()
	if r == dqStringEnd {
		lx.emit(itemKey)
		lx.next()
		return lexSkip(lx, lexKeyEnd)
	} else if r == eof {
		if lx.pos > lx.start {
			return lx.errorf("Unexpected EOF.")
		}
		lx.emit(itemEOF)
		return nil
	}
	lx.next()
	return lexDubQuotedKey
}

// lexQuotedKey consumes the text of a key between quotes.
func lexQuotedKey(lx *lexer) stateFn {
	r := lx.peek()
	if r == sqStringEnd {
		lx.emit(itemKey)
		lx.next()
		return lexSkip(lx, lexKeyEnd)
	} else if r == eof {
		if lx.pos > lx.start {
			return lx.errorf("Unexpected EOF.")
		}
		lx.emit(itemEOF)
		return nil
	}
	lx.next()
	return lexQuotedKey
}

// keyCheckKeyword will check for reserved keywords as the key value when the key is
// separated with a space.
func (lx *lexer) keyCheckKeyword(fallThrough, push stateFn) stateFn {
	key := strings.ToLower(lx.input[lx.start:lx.pos])
	switch key {
	case "include":
		lx.ignore()
		if push != nil {
			lx.push(push)
		}
		return lexIncludeStart
	}
	lx.emit(itemKey)
	return fallThrough
}

// lexIncludeStart will consume the whitespace til the start of the value.
func lexIncludeStart(lx *lexer) stateFn {
	r := lx.next()
	if isWhitespace(r) {
		return lexSkip(lx, lexIncludeStart)
	}
	lx.backup()
	return lexInclude
}

// lexIncludeQuotedString consumes the inner contents of a string. It assumes that the
// beginning '"' has already been consumed and ignored. It will not interpret any
// internal contents.
func lexIncludeQuotedString(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == sqStringEnd:
		lx.backup()
		lx.emit(itemInclude)
		lx.next()
		lx.ignore()
		return lx.pop()
	case r == eof:
		return lx.errorf("Unexpected EOF in quoted include")
	}
	return lexIncludeQuotedString
}

// lexIncludeDubQuotedString consumes the inner contents of a string. It assumes that the
// beginning '"' has already been consumed and ignored. It will not interpret any
// internal contents.
func lexIncludeDubQuotedString(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == dqStringEnd:
		lx.backup()
		lx.emit(itemInclude)
		lx.next()
		lx.ignore()
		return lx.pop()
	case r == eof:
		return lx.errorf("Unexpected EOF in double quoted include")
	}
	return lexIncludeDubQuotedString
}

// lexIncludeString consumes the inner contents of a raw string.
func lexIncludeString(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case isNL(r) || r == eof || r == optValTerm || r == mapEnd || isWhitespace(r):
		lx.backup()
		lx.emit(itemInclude)
		return lx.pop()
	case r == sqStringEnd:
		lx.backup()
		lx.emit(itemInclude)
		lx.next()
		lx.ignore()
		return lx.pop()
	}
	return lexIncludeString
}

// lexInclude will consume the include value.
func lexInclude(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == sqStringStart:
		lx.ignore() // ignore the " or '
		return lexIncludeQuotedString
	case r == dqStringStart:
		lx.ignore() // ignore the " or '
		return lexIncludeDubQuotedString
	case r == arrayStart:
		return lx.errorf("Expected include value but found start of an array")
	case r == mapStart:
		return lx.errorf("Expected include value but found start of a map")
	case r == blockStart:
		return lx.errorf("Expected include value but found start of a block")
	case unicode.IsDigit(r), r == '-':
		return lx.errorf("Expected include value but found start of a number")
	case r == '\\':
		return lx.errorf("Expected include value but found escape sequence")
	case isNL(r):
		return lx.errorf("Expected include value but found new line")
	}
	lx.backup()
	return lexIncludeString
}

// lexKey consumes the text of a key. Assumes that the first character (which
// is not whitespace) has already been consumed.
func lexKey(lx *lexer) stateFn {
	r := lx.peek()
	if unicode.IsSpace(r) {
		// Spaces signal we could be looking at a keyword, e.g. include.
		// Keywords will eat the keyword and set the appropriate return stateFn.
		return lx.keyCheckKeyword(lexKeyEnd, nil)
	} else if isKeySeparator(r) || r == eof {
		lx.emit(itemKey)
		return lexKeyEnd
	}
	lx.next()
	return lexKey
}

// lexKeyEnd consumes the end of a key (up to the key separator).
// Assumes that the first whitespace character after a key (or the '=' or ':'
// separator) has NOT been consumed.
func lexKeyEnd(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case unicode.IsSpace(r):
		return lexSkip(lx, lexKeyEnd)
	case isKeySeparator(r):
		return lexSkip(lx, lexValue)
	case r == eof:
		lx.emit(itemEOF)
		return nil
	}
	// We start the value here
	lx.backup()
	return lexValue
}

// lexValue starts the consumption of a value anywhere a value is expected.
// lexValue will ignore whitespace.
// After a value is lexed, the last state on the next is popped and returned.
func lexValue(lx *lexer) stateFn { panic("excised: lexValue") }

// lexArrayValue consumes one value in an array. It assumes that '[' or ','
// have already been consumed. All whitespace and new lines are ignored.
func lexArrayValue(lx *lexer) stateFn { panic("excised: lexArrayValue") }

// lexArrayValueEnd consumes the cruft between values of an array. Namely,
// it ignores whitespace and expects either a ',' or a ']'.
func lexArrayValueEnd(lx *lexer) stateFn { panic("excised: lexArrayValueEnd") }

// lexArrayEnd finishes the lexing of an array. It assumes that a ']' has
// just been consumed.
func lexArrayEnd(lx *lexer) stateFn { panic("excised: lexArrayEnd") }

// lexMapKeyStart consumes a key name up until the first non-whitespace
// character.
// lexMapKeyStart will ignore whitespace.
func lexMapKeyStart(lx *lexer) stateFn {
	r := lx.peek()
	switch {
	case isKeySeparator(r):
		return lx.errorf("Unexpected key separator '%v'.", r)
	case r == arrayEnd:
		return lx.errorf("Unexpected array end '%v' processing map.", r)
	case unicode.IsSpace(r):
		lx.next()
		return lexSkip(lx, lexMapKeyStart)
	case r == mapEnd:
		lx.next()
		return lexSkip(lx, lexMapEnd)
	case r == commentHashStart:
		lx.next()
		lx.push(lexMapKeyStart)
		return lexCommentStart
	case r == commentSlashStart:
		lx.next()
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexMapKeyStart)
			return lexCommentStart
		}
		lx.backup()
	case r == sqStringStart:
		lx.next()
		return lexSkip(lx, lexMapQuotedKey)
	case r == dqStringStart:
		lx.next()
		return lexSkip(lx, lexMapDubQuotedKey)
	case r == eof:
		return lx.errorf("Unexpected EOF processing map.")
	}
	lx.ignore()
	lx.next()
	return lexMapKey
}

// lexMapQuotedKey consumes the text of a key between quotes.
func lexMapQuotedKey(lx *lexer) stateFn {
	if r := lx.peek(); r == eof {
		return lx.errorf("Unexpected EOF processing quoted map key.")
	} else if r == sqStringEnd {
		lx.emit(itemKey)
		lx.next()
		return lexSkip(lx, lexMapKeyEnd)
	}
	lx.next()
	return lexMapQuotedKey
}

// lexMapDubQuotedKey consumes the text of a key between quotes.
func lexMapDubQuotedKey(lx *lexer) stateFn {
	if r := lx.peek(); r == eof {
		return lx.errorf("Unexpected EOF processing double quoted map key.")
	} else if r == dqStringEnd {
		lx.emit(itemKey)
		lx.next()
		return lexSkip(lx, lexMapKeyEnd)
	}
	lx.next()
	return lexMapDubQuotedKey
}

// lexMapKey consumes the text of a key. Assumes that the first character (which
// is not whitespace) has already been consumed.
func lexMapKey(lx *lexer) stateFn {
	if r := lx.peek(); r == eof {
		return lx.errorf("Unexpected EOF processing map key.")
	} else if unicode.IsSpace(r) {
		// Spaces signal we could be looking at a keyword, e.g. include.
		// Keywords will eat the keyword and set the appropriate return stateFn.
		return lx.keyCheckKeyword(lexMapKeyEnd, lexMapValueEnd)
	} else if isKeySeparator(r) {
		lx.emit(itemKey)
		return lexMapKeyEnd
	}
	lx.next()
	return lexMapKey
}

// lexMapKeyEnd consumes the end of a key (up to the key separator).
// Assumes that the first whitespace character after a key (or the '='
// separator) has NOT been consumed.
func lexMapKeyEnd(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case unicode.IsSpace(r):
		return lexSkip(lx, lexMapKeyEnd)
	case isKeySeparator(r):
		return lexSkip(lx, lexMapValue)
	}
	// We start the value here
	lx.backup()
	return lexMapValue
}

// lexMapValue consumes one value in a map. It assumes that '{' or ','
// have already been consumed. All whitespace and new lines are ignored.
// Map values can be separated by ',' or simple NLs.
func lexMapValue(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case unicode.IsSpace(r):
		return lexSkip(lx, lexMapValue)
	case r == mapValTerm:
		return lx.errorf("Unexpected map value terminator %q.", mapValTerm)
	case r == mapEnd:
		return lexSkip(lx, lexMapEnd)
	}
	lx.backup()
	lx.push(lexMapValueEnd)
	return lexValue
}

// lexMapValueEnd consumes the cruft between values of a map. Namely,
// it ignores whitespace and expects either a ',' or a '}'.
func lexMapValueEnd(lx *lexer) stateFn { panic("excised: lexMapValueEnd") }

// lexMapEnd finishes the lexing of a map. It assumes that a '}' has
// just been consumed.
func lexMapEnd(lx *lexer) stateFn {
	lx.ignore()
	lx.emit(itemMapEnd)
	return lx.pop()
}

// Checks if the unquoted string was actually a boolean
func (lx *lexer) isBool() bool { panic("excised: isBool") }

// Check if the unquoted string is a variable reference, starting with $.
func (lx *lexer) isVariable() bool { panic("excised: isVariable") }

// lexQuotedString consumes the inner contents of a string. It assumes that the
// beginning '"' has already been consumed and ignored. It will not interpret any
// internal contents.
func lexQuotedString(lx *lexer) stateFn { panic("excised: lexQuotedString") }

// lexDubQuotedString consumes the inner contents of a string. It assumes that the
// beginning '"' has already been consumed and ignored. It will not interpret any
// internal contents.
func lexDubQuotedString(lx *lexer) stateFn { panic("excised: lexDubQuotedString") }

// lexString consumes the inner contents of a raw string.
func lexString(lx *lexer) stateFn { panic("excised: lexString") }

// lexBlock consumes the inner contents as a string. It assumes that the
// beginning '(' has already been consumed and ignored. It will continue
// processing until it finds a ')' on a new line by itself.
func lexBlock(lx *lexer) stateFn { panic("excised: lexBlock") }

// lexStringEscape consumes an escaped character. It assumes that the preceding
// '\\' has already been consumed.
func lexStringEscape(lx *lexer) stateFn { panic("excised: lexStringEscape") }

// lexStringBinary consumes two hexadecimal digits following '\x'. It assumes
// that the '\x' has already been consumed.
func lexStringBinary(lx *lexer) stateFn { panic("excised: lexStringBinary") }

// lexNumberOrDateOrStringOrIPStart consumes either a (positive)
// integer, a float, a datetime, or IP, or String that started with a
// number.  It assumes that NO negative sign has been consumed, that
// is triggered above.
func lexNumberOrDateOrStringOrIPStart(lx *lexer) stateFn { panic("excised: lexNumberOrDateOrStringOrIPStart") }

// lexNumberOrDateOrStringOrIP consumes either a (positive) integer,
// float, datetime, IP or string without quotes that starts with a
// number.
func lexNumberOrDateOrStringOrIP(lx *lexer) stateFn { panic("excised: lexNumberOrDateOrStringOrIP") }

// lexConvenientNumber is when we have a suffix, e.g. 1k or 1Mb
func lexConvenientNumber(lx *lexer) stateFn { panic("excised: lexConvenientNumber") }

// lexDateAfterYear consumes a full Zulu Datetime in ISO8601 format.
// It assumes that "YYYY-" has already been consumed.
func lexDateAfterYear(lx *lexer) stateFn { panic("excised: lexDateAfterYear") }

// lexNegNumberStart consumes either an integer or a float. It assumes that a
// negative sign has already been read, but that *no* digits have been consumed.
// lexNegNumberStart will move to the appropriate integer or float states.
func lexNegNumberStart(lx *lexer) stateFn { panic("excised: lexNegNumberStart") }

// lexNegNumber consumes a negative integer or a float after seeing the first digit.
func lexNegNumber(lx *lexer) stateFn { panic("excised: lexNegNumber") }

// lexFloatStart starts the consumption of digits of a float after a '.'.
// Namely, at least one digit is required.
func lexFloatStart(lx *lexer) stateFn { panic("excised: lexFloatStart") }

// lexFloat consumes the digits of a float after a '.'.
// Assumes that one digit has been consumed after a '.' already.
func lexFloat(lx *lexer) stateFn { panic("excised: lexFloat") }

// lexIPAddr consumes IP addrs, like 127.0.0.1:4222
func lexIPAddr(lx *lexer) stateFn { panic("excised: lexIPAddr") }

// lexCommentStart begins the lexing of a comment. It will emit
// itemCommentStart and consume no characters, passing control to lexComment.
func lexCommentStart(lx *lexer) stateFn { panic("excised: lexCommentStart") }

// lexComment lexes an entire comment. It assumes that '#' has been consumed.
// It will consume *up to* the first new line character, and pass control
// back to the last state on the stack.
func lexComment(lx *lexer) stateFn { panic("excised: lexComment") }

// lexSkip ignores all slurped input and moves on to the next state.
func lexSkip(lx *lexer, nextState stateFn) stateFn {
	return func(lx *lexer) stateFn {
		lx.ignore()
		return nextState
	}
}

// Tests to see if we have a number suffix
func isNumberSuffix(r rune) bool {
	return r == 'k' || r == 'K' || r == 'm' || r == 'M' || r == 'g' || r == 'G' || r == 't' || r == 'T' || r == 'p' || r == 'P' || r == 'e' || r == 'E'
}

// Tests for both key separators
func isKeySeparator(r rune) bool {
	return r == keySepEqual || r == keySepColon
}

// isWhitespace returns true if `r` is a whitespace character according
// to the spec.
func isWhitespace(r rune) bool {
	return r == '\t' || r == ' '
}

func isNL(r rune) bool {
	return r == '\n' || r == '\r'
}

func (itype itemType) String() string {
	switch itype {
	case itemError:
		return "Error"
	case itemNIL:
		return "NIL"
	case itemEOF:
		return "EOF"
	case itemText:
		return "Text"
	case itemString:
		return "String"
	case itemBool:
		return "Bool"
	case itemInteger:
		return "Integer"
	case itemFloat:
		return "Float"
	case itemDatetime:
		return "DateTime"
	case itemKey:
		return "Key"
	case itemArrayStart:
		return "ArrayStart"
	case itemArrayEnd:
		return "ArrayEnd"
	case itemMapStart:
		return "MapStart"
	case itemMapEnd:
		return "MapEnd"
	case itemCommentStart:
		return "CommentStart"
	case itemVariable:
		return "Variable"
	case itemInclude:
		return "Include"
	}
	panic(fmt.Sprintf("BUG: Unknown type '%s'.", itype.String()))
}

func (item item) String() string {
	return fmt.Sprintf("(%s, '%s', %d, %d)", item.typ.String(), item.val, item.line, item.pos)
}

func escapeSpecial(c rune) string {
	switch c {
	case '\n':
		return "\\n"
	}
	return string(c)
}
