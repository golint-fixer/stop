package ast

import (
	"io"
	"math/big"

	"bitbucket.org/eaburns/stop/token"
)

// The Node interface is implemented by all nodes.
type Node interface {
	// Start returns the start location of the first token that induced
	// this node.
	Start() token.Location

	// End returns the end location of the final token that induced
	// this node.
	End() token.Location

	// Print writes a human-readable representation of the node
	// and the subtree beneath it to out.  If an error occurs then
	// it is panicked.
	print(level int, out io.Writer)

	// Dot writes the node and the subtree beneath it to a writer
	// in the dot language of graphviz.  If an error occurs then it
	// is panicked.
	dot(cur int, out io.Writer) int
}

// The Type interface is implemented by nodes that represent types.
type Type interface {
	Node
}

// An MapType is a type node that represents a map from types to types.
type MapType struct {
	Key    Type
	Type   Type
	mapLoc token.Location
}

func (n *MapType) Start() token.Location { return n.mapLoc }
func (n *MapType) End() token.Location   { return n.Type.End() }

// An ArrayType is a type node that represents an array of types.
type ArrayType struct {
	Size    Expression
	Type    Type
	openLoc token.Location
}

func (n *ArrayType) Start() token.Location { return n.openLoc }
func (n *ArrayType) End() token.Location   { return n.Type.End() }

// A SliceType is a type node that represents a slice of types.
type SliceType struct {
	Type    Type
	openLoc token.Location
}

func (n *SliceType) Start() token.Location { return n.openLoc }
func (n *SliceType) End() token.Location   { return n.Type.End() }

// A PointerType is a type node that represents a pointer to a type.
type PointerType struct {
	Type    Type
	starLoc token.Location
}

func (n *PointerType) Start() token.Location { return n.starLoc }
func (n *PointerType) End() token.Location   { return n.Type.End() }

// A TypeName is a type node that represents a named type.
type TypeName struct {
	Package string
	Name    string
	span
}

// The Expression interface is implemented by all nodes that are
// also expressions.
type Expression interface {
	Node
	// Loc returns a location that is indicative of this expression.
	// For example, the location of the operator of a binary
	// expression may be used.  Loc is used for error reporting.
	Loc() token.Location
}

// An Index is an expression node that represents indexing into an
// array or a slice.
type Index struct {
	Expression        Expression
	Index             Expression
	openLoc, closeLoc token.Location
}

func (n *Index) Start() token.Location { return n.Expression.Start() }
func (n *Index) Loc() token.Location   { return n.openLoc }
func (n *Index) End() token.Location   { return n.closeLoc }

// A Slice is an expression node that represents a slice of an array.
type Slice struct {
	Expression        Expression
	Low, High, Max    Expression
	openLoc, closeLoc token.Location
}

func (n *Slice) Start() token.Location { return n.Expression.Start() }
func (n *Slice) Loc() token.Location   { return n.openLoc }
func (n *Slice) End() token.Location   { return n.closeLoc }

// A TypeAssertion is an expression node representing a type assertion.
type TypeAssertion struct {
	Expression       Expression
	Type             Node
	dotLoc, closeLoc token.Location
}

func (n *TypeAssertion) Start() token.Location { return n.Expression.Start() }
func (n *TypeAssertion) Loc() token.Location   { return n.dotLoc }
func (n *TypeAssertion) End() token.Location   { return n.closeLoc }

// Selector is an expression node representing a selector or a
// qualified identifier.
type Selector struct {
	Expression Expression
	Selection  *Identifier
	dotLoc     token.Location
}

func (n *Selector) Start() token.Location { return n.Expression.Start() }
func (n *Selector) Loc() token.Location   { return n.dotLoc }
func (n *Selector) End() token.Location   { return n.Selection.End() }

// Call is a function call expression.
// After parsing but before type checking, a Call can represent
// either a function call or a type conversion.
type Call struct {
	Function  Expression
	Arguments []Expression
	// DotDotDot is true if the last argument ended with "...".
	DotDotDot         bool
	openLoc, closeLoc token.Location
}

func (c *Call) Start() token.Location { return c.Function.Start() }
func (c *Call) Loc() token.Location   { return c.openLoc }
func (c *Call) End() token.Location   { return c.closeLoc }

// BinaryOp is an expression node representing a binary operator.
type BinaryOp struct {
	Op          token.Token
	opLoc       token.Location
	Left, Right Expression
}

func (b *BinaryOp) Start() token.Location { return b.Left.Start() }
func (b *BinaryOp) Loc() token.Location   { return b.opLoc }
func (b *BinaryOp) End() token.Location   { return b.Right.End() }

// UnaryOp is an expression node representing a unary operator.
type UnaryOp struct {
	Op      token.Token
	opLoc   token.Location
	Operand Expression
}

func (u *UnaryOp) Start() token.Location { return u.opLoc }
func (u *UnaryOp) Loc() token.Location   { return u.opLoc }
func (u *UnaryOp) End() token.Location   { return u.Operand.End() }

// An Identifier is an expression node that represents an
// un-qualified identifier.
type Identifier struct {
	Name string
	span
}

// IntegerLiteral is an expression node representing a decimal,
// octal, or hex
// integer literal.
type IntegerLiteral struct {
	Value *big.Int
	span
}

// FloatLiteral is an expression node representing a floating point literal.
type FloatLiteral struct {
	Value *big.Rat
	span
}

// ImaginaryLiteral is an expression node representing an imaginary
// literal, the imaginary component of a complex number.
type ImaginaryLiteral struct {
	Value *big.Rat
	span
}

// RuneLiteral is an expression node representing a rune literal.
type RuneLiteral struct {
	Value rune
	span
}

// StringLiteral is an expression node representing an interpreted or
// raw string literal.
type StringLiteral struct {
	Value string
	span
}
