package lexer

import (
	"fmt"
)

// Punctuation tokens:
// [ ] ( ) { } . ->
// ++ -- & * + - ~ !
// / % << >> < > <= >= == != ^ | && ||
// ? : ; ...
// = *= /= %= += -= <<= >>= &= ^= |=
// , # ##
// <: :> <% %> %: %:%:

// punctuations that are always a single character:
// [ ] ( ) { } ~ ? ; ,

func (l *tokenizer) readPunctuation(firstByte byte) (Token, error) {
	if charClsassSinglePunctuation.contains(firstByte) {
		token := Token{
			Type:   TokenPunctuation,
			Value:  string(firstByte),
			Line:   l.line,
			Column: l.column,
		}
		l.column++
		return token, nil
	}

	buff := []byte{firstByte}

	switch firstByte {
	case '.':
		// . or ...
		return l.readDotOrEllipsis()
	case '-':
		// - or -> or -- or -=
		expectOneOfAndAppend(l.scanner, "->=", &buff)
	case '+':
		// + or ++ or +=
		expectOneOfAndAppend(l.scanner, "+=", &buff)
	case '&':
		// & or && or &=
		expectOneOfAndAppend(l.scanner, "&=", &buff)
	case '*':
		// * or *=
		expectOneOfAndAppend(l.scanner, "=", &buff)
	case '!':
		// ! or !=
		expectOneOfAndAppend(l.scanner, "=", &buff)
	case '/':
		// / or /= or comment, but comment is handled in readPunctuationOrComment, so we only need to check for /=
		expectOneOfAndAppend(l.scanner, "=", &buff)
	case '%':
		// % or %= or <% or %> or %: or %:%:
		return l.readPercentPunctuation()
	case '<':
		// < or <= or << or <<= or <: or <%
		expectOneOfAndAppend(l.scanner, "=<:%", &buff)
		if len(buff) == 2 && buff[1] == '<' {
			// potential for <<=, so we need to check for =
			expectOneOfAndAppend(l.scanner, "=", &buff)
		}
	case '>':
		// > or >= or >> or >>=
		expectOneOfAndAppend(l.scanner, "=>", &buff)
		if len(buff) == 2 && buff[1] == '>' {
			// potential for >>=, so we need to check for =
			expectOneOfAndAppend(l.scanner, "=", &buff)
		}
	case '^':
		// ^ or ^=
		expectOneOfAndAppend(l.scanner, "=", &buff)
	case ':':
		// : or :>
		expectOneOfAndAppend(l.scanner, ">", &buff)
	case '|':
		// | or || or |=
		expectOneOfAndAppend(l.scanner, "|=", &buff)
	case '=':
		// = or ==
		expectOneOfAndAppend(l.scanner, "=", &buff)
	case '#':
		// # or ##
		expectOneOfAndAppend(l.scanner, "#", &buff)

	default:
		return Token{Type: TokenError}, fmt.Errorf("unexpected charater %q at %d:%d", firstByte, l.line, l.column)
	}

	token := Token{
		Type:   TokenPunctuation,
		Value:  string(buff),
		Line:   l.line,
		Column: l.column,
	}
	l.column += len(buff)
	return token, nil
}

func (l *tokenizer) readDotOrEllipsis() (Token, error) {
	buff := []byte{'.'}
	err := expectOneOfAndAppend(l.scanner, ".", &buff)
	if err != nil {
		return Token{}, err
	}
	if len(buff) == 1 {
		// single dot
		l.column++
		return Token{
			Type:   TokenPunctuation,
			Value:  ".",
			Line:   l.line,
			Column: l.column - 1,
		}, nil
	}

	// we got .. so we need to read the third dot or unread the second dot
	err = expectOneOfAndAppend(l.scanner, ".", &buff)
	if err != nil {
		return Token{}, err
	}
	if len(buff) == 2 {
		// we got .. but not ... so we need to unread the second dot
		l.scanner.UnreadByte()
		l.column++
		return Token{
			Type:   TokenPunctuation,
			Value:  ".",
			Line:   l.line,
			Column: l.column - 1,
		}, nil
	}

	token := Token{
		Type:   TokenPunctuation,
		Value:  string(buff),
		Line:   l.line,
		Column: l.column,
	}
	l.column += len(buff)
	return token, nil
}

func (l *tokenizer) readPercentPunctuation() (Token, error) {
	// % or %= or <% or %> or %: or %:%:
	buff := []byte{'%'}
	err := expectOneOfAndAppend(l.scanner, "=:<>", &buff)
	if err != nil {
		return Token{}, err
	}
	if len(buff) == 1 {
		// single %
		l.column++
		return Token{
			Type:   TokenPunctuation,
			Value:  "%",
			Line:   l.line,
			Column: l.column - 1,
		}, nil
	}

	// if we don't get %: no need to check for %:%:
	if buff[1] != ':' {
		token := Token{
			Type:   TokenPunctuation,
			Value:  string(buff),
			Line:   l.line,
			Column: l.column,
		}
		l.column += len(buff)
		return token, nil
	}

	// %: or %:%:
	err = expectOneOfAndAppend(l.scanner, "%", &buff)
	if err != nil {
		return Token{}, err
	}
	if len(buff) == 3 {
		// got %:%, now ':' or unread the third character
		err = expectOneOfAndAppend(l.scanner, ":", &buff)
		if err != nil {
			return Token{}, err
		}
		if len(buff) == 3 {
			// only %:% so we need to unread the third character
			l.scanner.UnreadByte()
			l.column += 2
			return Token{
				Type:   TokenPunctuation,
				Value:  "%:",
				Line:   l.line,
				Column: l.column - 2,
			}, nil
		}
	}

	token := Token{
		Type:   TokenPunctuation,
		Value:  string(buff),
		Line:   l.line,
		Column: l.column,
	}
	l.column += len(buff)
	return token, nil
}
