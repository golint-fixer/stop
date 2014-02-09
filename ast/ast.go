package ast

import (
	"io"
	"math/big"

	"github.com/velour/stop/token"
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

// A Declaration is a node representing a declaration.
type Declaration interface {
	Node

	// Comments returns the comments appearing before this
	// declaration without an intervening blank line.
	Comments() []string
}

// A Comments implements the Comments method of the Declaration
// and Statements interfaces.
type comments []string

func (c comments) Comments() []string {
	return []string(c)
}

// A Declarations is node representing a non-empty list of declarations.
type Declarations []Declaration

func (n Declarations) Start() token.Location {
	return n[0].Start()
}

func (n Declarations) End() token.Location {
	return n[len(n)-1].End()
}

// A ConstSpec is a declaration node representing the declaration of
// a series of constants.
type ConstSpec struct {
	comments
	// Type is the type of the spec or nil if the type should be inferred
	// from the values.
	Type   Type
	Names  []Identifier
	Values []Expression
}

func (n *ConstSpec) Start() token.Location {
	return n.Names[0].Start()
}

func (n *ConstSpec) End() token.Location {
	return n.Values[len(n.Values)-1].End()
}

// A TypeSpec is a declaration node representing the declaration of
// a single type.
type TypeSpec struct {
	comments
	Name Identifier
	Type Type
}

func (n *TypeSpec) Start() token.Location {
	return n.Name.Start()
}

func (n *TypeSpec) End() token.Location {
	return n.Type.End()
}

// The Type interface is implemented by nodes that represent types.
type Type interface {
	Node
}

// A StructType is a type node representing a struct type.
type StructType struct {
	Fields               []FieldDecl
	keywordLoc, closeLoc token.Location
}

func (n *StructType) Start() token.Location {
	return n.keywordLoc
}

func (n *StructType) End() token.Location {
	return n.closeLoc
}

// A FieldDecl is a node representing a struct field declaration.
type FieldDecl struct {
	Identifiers []Identifier
	Type        Type
	Tag         *StringLiteral
}

func (n *FieldDecl) Start() token.Location {
	if len(n.Identifiers) > 0 {
		return n.Identifiers[0].Start()
	}
	return n.Type.Start()
}

func (n *FieldDecl) End() token.Location {
	if n.Tag != nil {
		return n.Tag.End()
	}
	return n.Type.End()
}

// An InterfaceType is a type node representing an interface type.
type InterfaceType struct {
	// Methods is a slice of the methods declarations of this interface.
	// A method declaration is either a Method, giving the name and
	// signature of a method, or a TypeName, naming an interface
	// whose methods are included in this interface too.
	Methods              []Node
	keywordLoc, closeLoc token.Location
}

func (n *InterfaceType) Start() token.Location {
	return n.keywordLoc
}

func (n *InterfaceType) End() token.Location {
	return n.closeLoc
}

// A Method is a node representing a method name and its signature.
type Method struct {
	Name Identifier
	Signature
}

func (n *Method) Start() token.Location {
	return n.Name.Start()
}

func (n *Method) End() token.Location {
	return n.Signature.End()
}

// A FunctionType is a type node representing a function type.
type FunctionType struct {
	Signature
}

// A Signature is a node representing a parameter list and result types.
type Signature struct {
	Parameters ParameterList
	Result     ParameterList
}

func (n *Signature) Start() token.Location { return n.Parameters.Start() }
func (n *Signature) End() token.Location   { return n.Result.End() }

// A ParameterList is a node representing a, possibly empty,
// parenthesized list of parameters.
type ParameterList struct {
	Parameters        []ParameterDecl
	openLoc, closeLoc token.Location
}

func (n *ParameterList) Start() token.Location { return n.openLoc }
func (n *ParameterList) End() token.Location   { return n.closeLoc }

// A ParameterDecl is a node representing the declaration of a set
// of parameters of a common type.
type ParameterDecl struct {
	Type        Type
	Identifiers []Identifier
	// DotDotDot is true if the final identifier was followed by a "...".
	DotDotDot bool
}

func (n *ParameterDecl) Start() token.Location {
	if len(n.Identifiers) > 0 {
		return n.Identifiers[0].Start()
	}
	return n.Type.Start()
}
func (n *ParameterDecl) End() token.Location { return n.Type.End() }

// A ChannelType is a type node that represents a send, receive, or
// a send and receive channel.
type ChannelType struct {
	Send, Receive bool
	Type          Type
	startLoc      token.Location
}

func (n *ChannelType) Start() token.Location { return n.startLoc }
func (n *ChannelType) End() token.Location   { return n.Type.End() }

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
	// If size==nil then this is an array type for a composite literal
	// with the size specified using [...]Type notation.
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

// A CompositeLiteral is an expression node that represents a
// composite literal.
type CompositeLiteral struct {
	// Type may be nil.
	Type              Type
	Elements          []Element
	openLoc, closeLoc token.Location
}

func (n *CompositeLiteral) Start() token.Location {
	if n.Type != nil {
		return n.openLoc
	}
	return n.Type.Start()
}

func (n *CompositeLiteral) Loc() token.Location {
	return n.Start()
}

func (n *CompositeLiteral) End() token.Location {
	return n.closeLoc
}

// An Element is a node representing the key-value mapping
// of a single element of a composite literal.
type Element struct {
	Key   Expression
	Value Expression
}

func (n *Element) Start() token.Location {
	if n.Key != nil {
		return n.Key.Start()
	}
	return n.Value.Start()
}

func (n *Element) End() token.Location {
	return n.Value.End()
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
