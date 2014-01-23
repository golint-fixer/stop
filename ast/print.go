package ast

import (
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"
)

// PrintFloatPrecision is the number of digits of precision to use when
// printing floating point values.  Trailing zeroes are trimmed, so
// using a large value won't necessarily lead to extremely large strings.
const printFloatPrecision = 20

// Print writes a human-readable representation of an abstract
// syntax tree to an io.Writer.
func Print(out io.Writer, n Node) (err error) {
	defer recoverError(&err)
	n.print(0, out)
	return
}

func (n *MapType) print(level int, out io.Writer) {
	format(out, level, "MapType{\n\tKey: ")
	n.Key.print(level+1, out)
	format(out, level, "\n\tType: ")
	n.Type.print(level+1, out)
	format(out, level, "\n}")
}

func (n *ArrayType) print(level int, out io.Writer) {
	format(out, level, "ArrayType{\n\tSize: ")
	n.Size.print(level+1, out)
	format(out, level, "\n\tType: ")
	n.Type.print(level+1, out)
	format(out, level, "\n}")
}

func (n *SliceType) print(level int, out io.Writer) {
	format(out, level, "SliceType{\n\tType: ")
	n.Type.print(level+1, out)
	format(out, level, "\n}")
}

func (n *PointerType) print(level int, out io.Writer) {
	format(out, level, "PointerType{\n\tType: ")
	n.Type.print(level+1, out)
	format(out, level, "\n}")
}

func (n *TypeName) print(level int, out io.Writer) {
	format(out, level, "TypeName{\n\tPackage: %s\n\tName: %s\n}",
		n.Package, n.Name)
}

func (n *Index) print(level int, out io.Writer) {
	format(out, level, "Index{\n\tExpression: ")
	n.Expression.print(level+1, out)
	format(out, level, "\n\tIndex: ")
	n.Index.print(level+1, out)
	format(out, level, "\n}")
}

func (n *Slice) print(level int, out io.Writer) {
	format(out, level, "Slice{\n\tExpression: ")
	n.Expression.print(level+1, out)
	format(out, level, "\n\tLow: ")
	if n.Low != nil {
		n.Low.print(level+1, out)
	} else {
		io.WriteString(out, "nil")
	}
	format(out, level, "\n\tHigh: ")
	if n.High != nil {
		n.High.print(level+1, out)
	} else {
		io.WriteString(out, "nil")
	}
	format(out, level, "\n\tMax: ")
	if n.Max != nil {
		n.Max.print(level+1, out)
	} else {
		io.WriteString(out, "nil")
	}
	format(out, level, "\n}")
}

func (n *Call) print(level int, out io.Writer) {
	format(out, level, "Call{\n\tFunction: ")
	n.Function.print(level+1, out)
	format(out, level, "\n\tArguments: [")
	indent := "\n" + strings.Repeat("\t", level+2)
	for _, e := range n.Arguments {
		io.WriteString(out, indent)
		e.print(level+2, out)
	}
	format(out, level, "\n\t]\n\tDotDotDot: %t\n}", n.DotDotDot)
}

func (n *TypeAssertion) print(level int, out io.Writer) {
	format(out, level, "TypeAssertion{\n\tExpression: ")
	n.Expression.print(level+1, out)
	format(out, level, "\n\tType: ")
	n.Type.print(level+1, out)
	format(out, level, "\n}")
}

func (n *Selector) print(level int, out io.Writer) {
	format(out, level, "Selector{\n\tExpression: ")
	n.Expression.print(level+1, out)
	format(out, level, "\n\tSelection: ")
	n.Selection.print(level+1, out)
	format(out, level, "\n}")
}

func (n *BinaryOp) print(level int, out io.Writer) {
	format(out, level, "BinaryOp{\n\tOp: %s\n\tLeft: ", n.Op.String())
	n.Left.print(level+1, out)
	format(out, level, "\n\tRight: ")
	n.Right.print(level+1, out)
	format(out, level, "\n}")
}

func (n *UnaryOp) print(level int, out io.Writer) {
	format(out, level, "UnaryOp{\n\tOp: %s\n\tOperand: ", n.Op.String())
	n.Operand.print(level+1, out)
	format(out, level, "\n}")
}

func (n *IntegerLiteral) print(level int, out io.Writer) {
	format(out, level, "IntegerLiteral{ Value: %s }", n.Value.String())
}

func (n *FloatLiteral) print(level int, out io.Writer) {
	format(out, level, "FloatLiteral{ Value: %s }", n.printString())
}

func (n *FloatLiteral) printString() string {
	return ratPrintString(n.Value)
}

func (n *ImaginaryLiteral) print(level int, out io.Writer) {
	format(out, level, "ImaginaryLiteral{ Value: %s }", n.printString())
}

func (n *ImaginaryLiteral) printString() string {
	return ratPrintString(n.Value) + "i"
}

func (n *RuneLiteral) print(level int, out io.Writer) {
	format(out, level, "RuneLiteral{ Value: %s (0x%x) }", strconv.QuoteRune(n.Value), n.Value)
}

func (n *StringLiteral) print(level int, out io.Writer) {
	format(out, level, "StringLiteral{ Value: %s }", n.Value)
}

func (n *Identifier) print(level int, out io.Writer) {
	format(out, level, "Identifier{ Name: %s }", n.Name)
}

// Format writes a formatted string in the style of fmt.Fprintf to out.
// Trailing newlines are stripped from the format string, form, and
// all remaining newlines are suffixed with level tab characters.  If
// an error occurs, it is panicked.
func format(out io.Writer, level int, form string, args ...interface{}) {
	form = strings.TrimRight(form, "\n")
	indent := strings.Repeat("\t", level)
	form = strings.Replace(form, "\n", "\n"+indent, -1)
	if _, err := fmt.Fprintf(out, form, args...); err != nil {
		panic(err)
	}
}

// RecoverError recovers a panick.  If the recovered value implements
// the error interface then it is assigned to the pointee of err.
func recoverError(err *error) {
	r := recover()
	if r == nil {
		return
	}
	if e, ok := r.(error); ok {
		*err = e
	}
}

// RatPrintString returns a printable string representation of a big.Rat.
func ratPrintString(rat *big.Rat) string {
	s := rat.FloatString(printFloatPrecision)
	s = strings.TrimRight(s, "0")
	if len(s) > 0 && s[len(s)-1] == '.' {
		s += "0"
	}
	return s
}
