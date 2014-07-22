package ast

import (
	"fmt"
	"strconv"

	"github.com/velour/stop/token"
)

// Errors defines a slice of errors. The Error method returns the concatination of the
// Error strings of all errors, separated by newlines.
type errors []error

func (es errors) Error() string {
	s := ""
	for _, e := range es {
		if s != "" {
			s += "\n"
		}
		s += e.Error()
	}
	return s
}

// Errs returns an errors from a sequence of errors.
func errs(es ...error) errors {
	return errors(es)
}

// Add adds an error to the errors if it is non-nil.
func (es *errors) Add(e error) {
	if e != nil {
		*es = append(*es, e)
	}
}

// ErrorOrNil returns nil if the errors is empty or it returns the errors as an error.
func (es errors) ErrorOrNil() error {
	if len(es) == 0 {
		return nil
	}
	return es
}

// A SyntaxError is an error that describes a parse failure: something
// unexpected in the syntax of the Go source code.
type SyntaxError struct {
	// Wanted describes the value that was expected.
	Wanted string
	// Got describes the unexpected token.
	Got fmt.Stringer
	// Text is the text of the unexpected token.
	Text string
	// Start and End give the location of the error.
	Start, End token.Location
	// Stack is a human-readable trace of the stack showing the
	// cause of the syntax error.
	Stack string
}

func (e *SyntaxError) Error() string {
	if e.Got == token.Error {
		text := strconv.QuoteToASCII(e.Text)
		return fmt.Sprintf("%s: unexpected rune in input [%s]", e.Start, text[1:len(text)-1])
	}
	switch e.Got {
	case token.Semicolon:
		e.Text = ";"
	case token.EOF:
		e.Text = "EOF"
	}
	return fmt.Sprintf("%s: expected %s, got %s", e.Start, e.Wanted, e.Text)
}

// A MalformedLiteral is an error that describes a literal for which the
// value was malformed.  For example, an integer literal for which the
// text is not parsable as an integer.
//
// BUG(eaburns): The parser should not have to deal with this class
// of error.  The lexer should return an Error token if it scans a literal
// that is lexicographically incorrect.
type MalformedLiteral struct {
	// The name for the literal type.
	Type string
	// The malformed text.
	Text string
	// Start and End give the location of the error.
	Start, End token.Location
}

func (e *MalformedLiteral) Error() string {
	return fmt.Sprintf("%s: malformed %s: [%s]", e.Start, e.Type, e.Text)
}

// Redeclaration is an error that denotes multiple definitions of the same
// variable within the same scope.
type Redeclaration struct {
	Name          string
	First, Second Declaration
}

func (e *Redeclaration) Error() string {
	return fmt.Sprintf("%s: %s redeclared, originally declared at %s", e.Second.Start(), e.Name, e.First.Start())
}