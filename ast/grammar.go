package ast

import (
	"math/big"
	"strconv"
	"strings"

	"bitbucket.org/eaburns/stop/token"
)

var (
	// Binary op precedence for precedence climbing algorithm.
	// http://www.engr.mun.ca/~theo/Misc/exp_parsing.htm
	//
	// BUG(eaburns): Define tokens.NTokens and change
	// map[token.Token]Whatever to [nTokens]Whatever.
	precedence = map[token.Token]int{
		token.OrOr:           1,
		token.AndAnd:         2,
		token.EqualEqual:     3,
		token.BangEqual:      3,
		token.Less:           3,
		token.LessEqual:      3,
		token.Greater:        3,
		token.GreaterEqual:   3,
		token.Plus:           4,
		token.Minus:          4,
		token.Or:             4,
		token.Carrot:         4,
		token.Star:           5,
		token.Divide:         5,
		token.Percent:        5,
		token.LessLess:       5,
		token.GreaterGreater: 5,
		token.And:            5,
		token.AndCarrot:      5,
	}

	// Set of unary operators.
	unary = map[token.Token]bool{
		token.Plus:      true,
		token.Minus:     true,
		token.Bang:      true,
		token.Carrot:    true,
		token.Star:      true,
		token.And:       true,
		token.LessMinus: true,
	}
)

// Parse returns the root of an abstract syntax tree for the Go language
// or an error if one is encountered.
//
// BUG(eaburns): This is currently just for testing since it doesn't
// parse the top-level production.
func Parse(p *Parser) (root Node, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		switch e := r.(type) {
		case *SyntaxError:
			err = e
		case *MalformedLiteral:
			err = e
		default:
			panic(r)
		}

	}()
	root = parseExpression(p)
	return
}

// Set of tokens that start a type.
var typeFirst = map[token.Token]bool{
	token.Identifier:  true,
	token.Star:        true,
	token.OpenBracket: true,
	token.Struct:      true,
	token.Func:        true,
	token.Interface:   true,
	token.Map:         true,
	token.Chan:        true,
	token.LessMinus:   true,
	token.OpenParen:   true,
}

// BUG(eaburns): Types are incomplete.
func parseType(p *Parser) Type {
	switch p.tok {
	case token.Identifier:
		return parseTypeName(p)

	case token.Star:
		starLoc := p.lex.Start
		p.next()
		return &PointerType{Type: parseType(p), starLoc: starLoc}

	case token.OpenBracket:
		return parseArrayOrSliceType(p)

	case token.Struct:
		panic("unimplemented")

	case token.Func:
		p.next()
		return &FunctionType{parseSignature(p)}

	case token.Interface:
		panic("unimplemented")

	case token.Map:
		return parseMapType(p)

	case token.Chan:
		fallthrough
	case token.LessMinus:
		return parseChannelType(p)

	case token.OpenParen:
		p.next()
		t := parseType(p)
		p.expect(token.CloseParen)
		p.next()
		return t
	}

	panic(p.err(token.Identifier, token.Star, token.OpenBracket,
		token.Struct, token.Func, token.Interface, token.Map,
		token.Chan, token.LessMinus, token.OpenParen))
}

func parseSignature(p *Parser) Signature {
	s := Signature{Parameters: parseParameterList(p)}
	if p.tok == token.OpenParen {
		results := parseParameterList(p)
		s.Result = &results
	} else if typeFirst[p.tok] {
		s.Result = parseType(p)
	}
	return s
}

// Parsing a parameter list is a bit complex.  The grammar productions
// in the spec are more permissive than the language actually allows.
// The text of the spec restricts parameter lists to be either a series of
// parameter declarations without identifiers and only types, or a
// series of declarations that all have one or more identifiers.  Instead,
// we use the grammar below, which only allows one type of list or the
// other, but not both.
//
// ParameterList = "(" ParameterListTail
// ParameterListTail =
// 	| “)”
// 	| Identifier “,” ParameterListTail
// 	| Identifier “.” Identifier TypeParameterList
// 	| Identifier Type DeclParameterList
// 	| NonTypeNameType TypeParameterList
// 	| “...” Type ")"
// TypeParameterList =
// 	| “)”
// 	| "," ")"
// 	| “,” Type TypeParameterList
// 	| “,” “...” Type ")"
// DeclParameterList =
// 	| ")"
// 	| "," ")"
// 	| "," IdentifierList Type DeclParameterList
// 	| "," IdentifierList "..." Type ")"
// IdentifierList =
// 	| Identifier “,” IdentifierList
// 	| Identifier
func parseParameterList(p *Parser) ParameterList {
	p.expect(token.OpenParen)
	pl := ParameterList{openLoc: p.lex.Start}
	p.next()
	parseParameterListTail(p, &pl, nil)
	p.expect(token.CloseParen)
	pl.closeLoc = p.lex.Start
	p.next()
	return pl
}

func parseParameterListTail(p *Parser, pl *ParameterList, idents []Identifier) {
	switch {
	case p.tok == token.CloseParen:
		pl.closeLoc = p.lex.Start
		pl.Parameters = typeNameDecls(idents)
		return

	case p.tok == token.Identifier:
		id := parseIdentifier(p)
		switch {
		case p.tok == token.Comma:
			p.next()
			fallthrough
		case p.tok == token.CloseParen:
			idents = append(idents, *id)
			parseParameterListTail(p, pl, idents)
			return

		case p.tok == token.Dot:
			p.next()
			p.expect(token.Identifier)
			t := &TypeName{
				Package: id.Name,
				Name:    p.text(),
				span:    span{start: id.span.start, end: p.lex.End},
			}
			p.next()
			d := ParameterDecl{Type: t}
			pl.Parameters = append(typeNameDecls(idents), d)
			parseTypeParameterList(p, pl)
			return

		default:
			idents = append(idents, *id)
			d := ParameterDecl{Identifiers: idents}
			if p.tok == token.DotDotDot {
				d.DotDotDot = true
				p.next()
			}
			d.Type = parseType(p)
			pl.Parameters = []ParameterDecl{d}
			if !d.DotDotDot {
				parseDeclParameterList(p, pl)
			}
			return
		}

	case p.tok == token.DotDotDot:
		p.next()
		d := ParameterDecl{Type: parseType(p), DotDotDot: true}
		pl.Parameters = append(typeNameDecls(idents), d)
		return

	case typeFirst[p.tok]:
		d := ParameterDecl{Type: parseType(p)}
		pl.Parameters = append(typeNameDecls(idents), d)
		parseTypeParameterList(p, pl)
		return
	}

	panic(p.err(")", "...", "identifier", "type"))
}

func parseTypeParameterList(p *Parser, pl *ParameterList) {
	if p.tok == token.CloseParen {
		return
	}
	p.expect(token.Comma)
	p.next()

	// Allow trailing comma.
	if p.tok == token.CloseParen {
		return
	}

	d := ParameterDecl{}
	if p.tok == token.DotDotDot {
		d.DotDotDot = true
		p.next()
	}
	d.Type = parseType(p)
	pl.Parameters = append(pl.Parameters, d)
	if !d.DotDotDot {
		parseTypeParameterList(p, pl)
	}
	return
}

func parseDeclParameterList(p *Parser, pl *ParameterList) {
	if p.tok == token.CloseParen {
		return
	}

	p.expect(token.Comma)
	p.next()

	// Allow trailing comma.
	if p.tok == token.CloseParen {
		return
	}

	d := ParameterDecl{}
	for {
		id := parseIdentifier(p)
		d.Identifiers = append(d.Identifiers, *id)

		if p.tok != token.Comma {
			break
		}
		p.next()
	}
	if p.tok == token.DotDotDot {
		d.DotDotDot = true
		p.next()
	}
	d.Type = parseType(p)
	pl.Parameters = append(pl.Parameters, d)
	parseDeclParameterList(p, pl)
	return
}

func typeNameDecls(idents []Identifier) []ParameterDecl {
	decls := make([]ParameterDecl, len(idents))
	for i, id := range idents {
		decls[i].Type = &TypeName{Name: id.Name, span: id.span}
	}
	return decls
}

func parseChannelType(p *Parser) Type {
	ch := &ChannelType{Send: true, Receive: true, startLoc: p.lex.Start}
	if p.tok == token.LessMinus {
		ch.Send = false
		p.next()
	}
	p.expect(token.Chan)
	p.next()
	if ch.Send && p.tok == token.LessMinus {
		ch.Receive = false
		p.next()
	}
	ch.Type = parseType(p)
	return ch
}

func parseMapType(p *Parser) Type {
	p.expect(token.Map)
	m := &MapType{mapLoc: p.lex.Start}
	p.next()
	p.expect(token.OpenBracket)
	p.next()
	m.Key = parseType(p)
	p.expect(token.CloseBracket)
	p.next()
	m.Type = parseType(p)
	return m
}

func parseArrayOrSliceType(p *Parser) Type {
	p.expect(token.OpenBracket)
	openLoc := p.lex.Start
	p.next()

	if p.tok == token.CloseBracket {
		p.next()
		sl := &SliceType{Type: parseType(p), openLoc: openLoc}
		return sl
	}

	ar := &ArrayType{Size: parseExpression(p), openLoc: openLoc}
	p.expect(token.CloseBracket)
	p.next()
	ar.Type = parseType(p)
	return ar
}

func parseTypeName(p *Parser) Type {
	p.expect(token.Identifier)
	n := &TypeName{Name: p.text(), span: p.span()}
	p.next()
	if p.tok == token.Dot {
		p.next()
		p.expect(token.Identifier)
		n.Package = n.Name
		n.Name = p.text()
		p.next()
	}
	return n
}

func parseExpression(p *Parser) Expression {
	return parseBinaryExpr(p, 1)
}

func parseBinaryExpr(p *Parser, prec int) Expression {
	left := parseUnaryExpr(p)
	for {
		pr, ok := precedence[p.tok]
		if !ok || pr < prec {
			return left
		}
		op, opLoc := p.tok, p.lex.Start
		p.next()
		right := parseBinaryExpr(p, pr+1)
		left = &BinaryOp{
			Op:    op,
			opLoc: opLoc,
			Left:  left,
			Right: right,
		}
	}
}

func parseUnaryExpr(p *Parser) Expression {
	if unary[p.tok] {
		op, opLoc := p.tok, p.lex.Start
		p.next()
		operand := parseUnaryExpr(p)
		return &UnaryOp{
			Op:      op,
			opLoc:   opLoc,
			Operand: operand,
		}
	}
	return parsePrimaryExpr(p)
}

func parsePrimaryExpr(p *Parser) Expression {
	left := parseOperand(p)
	for {
		switch p.tok {
		case token.OpenBracket:
			left = parseSliceOrIndex(p, left)
		case token.OpenParen:
			left = parseCall(p, left)
		case token.Dot:
			left = parseSelectorOrTypeAssertion(p, left)
		default:
			return left
		}
	}
}

func parseSliceOrIndex(p *Parser, left Expression) Expression {
	p.expect(token.OpenBracket)
	openLoc := p.lex.Start
	p.next()

	if p.tok == token.Colon {
		sl := parseSliceHighMax(p, left, nil)
		sl.openLoc = openLoc
		return sl
	}

	e := parseExpression(p)

	switch p.tok {
	case token.CloseBracket:
		index := &Index{Expression: left, Index: e, openLoc: openLoc}
		p.expect(token.CloseBracket)
		index.closeLoc = p.lex.End
		p.next()
		return index

	case token.Colon:
		sl := parseSliceHighMax(p, left, e)
		sl.openLoc = openLoc
		return sl
	}

	panic(p.err(token.CloseBracket, token.Colon))
}

// Parses the remainder of a slice expression, beginning from the
// colon after the low term of the expression.  The returned Slice
// node does not have its openLoc field set; it must be set by the
// caller.
func parseSliceHighMax(p *Parser, left, low Expression) *Slice {
	p.expect(token.Colon)
	p.next()

	sl := &Slice{Expression: left, Low: low}
	if p.tok != token.CloseBracket {
		sl.High = parseExpression(p)
		if p.tok == token.Colon {
			p.next()
			sl.Max = parseExpression(p)
		}
	}
	p.expect(token.CloseBracket)
	sl.closeLoc = p.lex.Start
	p.next()
	return sl
}

func parseCall(p *Parser, left Expression) Expression {
	p.expect(token.OpenParen)
	c := &Call{Function: left, openLoc: p.lex.Start}
	p.next()
	c.Arguments = parseExpressionList(p)
	if p.tok == token.DotDotDot {
		c.DotDotDot = true
		p.next()
	}
	p.expect(token.CloseParen)
	c.closeLoc = p.lex.End
	p.next()
	return c
}

func parseExpressionList(p *Parser) []Expression {
	var exprs []Expression
	for {
		exprs = append(exprs, parseExpression(p))
		if p.tok != token.Comma {
			break
		}
		p.next()
	}
	return exprs
}

func parseOperand(p *Parser) Expression {
	switch p.tok {
	case token.Identifier:
		id := parseIdentifier(p)
		if p.tok == token.Dot {
			return parseSelectorOrTypeAssertion(p, id)
		}
		return id

	case token.IntegerLiteral:
		return parseIntegerLiteral(p)

	case token.FloatLiteral:
		return parseFloatLiteral(p)

	case token.ImaginaryLiteral:
		return parseImaginaryLiteral(p)

	case token.RuneLiteral:
		return parseRuneLiteral(p)

	case token.StringLiteral:
		return parseStringLiteral(p)

	// BUG(eaburns): Composite literal
	// BUG(eaburns): Function literal

	case token.OpenParen:
		p.next()
		e := parseExpression(p)
		p.expect(token.CloseParen)
		p.next()
		return e

	default:
		panic(p.err("operand"))
	}
}

func parseSelectorOrTypeAssertion(p *Parser, left Expression) Expression {
	p.expect(token.Dot)
	dotLoc := p.lex.Start
	p.next()

	switch p.tok {
	case token.OpenParen:
		p.next()
		t := &TypeAssertion{Expression: left, dotLoc: dotLoc}
		t.Type = parseType(p)
		p.expect(token.CloseParen)
		t.closeLoc = p.lex.Start
		p.next()
		return t

	case token.Identifier:
		left = &Selector{
			Expression: left,
			Selection:  parseIdentifier(p),
			dotLoc:     dotLoc,
		}
		if p.tok == token.Dot {
			return parseSelectorOrTypeAssertion(p, left)
		}
		return left
	}

	panic(p.err(token.OpenParen, token.Identifier))
}

func parseIntegerLiteral(p *Parser) Expression {
	l := &IntegerLiteral{Value: new(big.Int), span: p.span()}
	if _, ok := l.Value.SetString(p.lex.Text(), 0); ok {
		p.next()
		return l
	}
	// This check is needed to catch malformed octal literals that
	// are currently allowed by the lexer.  For example, 08.
	panic(&MalformedLiteral{
		Type:  "integer literal",
		Text:  p.text(),
		Start: p.lex.Start,
		End:   p.lex.End,
	})
}

func parseFloatLiteral(p *Parser) Expression {
	l := &FloatLiteral{Value: new(big.Rat), span: p.span()}
	if _, ok := l.Value.SetString(p.lex.Text()); ok {
		p.next()
		return l
	}
	// I seem to recall that there was some case where the lexer
	// may return a malformed float, but I can't remember the
	// specifics.
	panic(&MalformedLiteral{
		Type:  "float literal",
		Text:  p.text(),
		Start: p.lex.Start,
		End:   p.lex.End,
	})
}

func parseImaginaryLiteral(p *Parser) Expression {
	text := p.lex.Text()
	if len(text) < 1 || text[len(text)-1] != 'i' {
		panic("bad imaginary literal: " + text)
	}
	text = text[:len(text)-1]
	l := &ImaginaryLiteral{Value: new(big.Rat), span: p.span()}
	if _, ok := l.Value.SetString(text); ok {
		p.next()
		return l
	}

	// I seem to recall that there was some case where the lexer
	// may return a malformed float, but I can't remember the
	// specifics.
	panic(&MalformedLiteral{
		Type:  "imaginary literal",
		Text:  p.text(),
		Start: p.lex.Start,
		End:   p.lex.End,
	})
}

func parseStringLiteral(p *Parser) Expression {
	text := p.lex.Text()
	if len(text) < 2 {
		panic("bad string literal: " + text)
	}
	l := &StringLiteral{span: p.span()}
	if text[0] == '`' {
		l.Value = strings.Replace(text[1:len(text)-1], "\r", "", -1)
	} else {
		var err error
		if l.Value, err = strconv.Unquote(text); err != nil {
			panic("bad string literal: " + p.lex.Text())
		}
	}
	p.next()
	return l
}

func parseIdentifier(p *Parser) *Identifier {
	p.expect(token.Identifier)
	id := &Identifier{Name: p.text(), span: p.span()}
	p.next()
	return id
}

func parseRuneLiteral(p *Parser) Expression {
	text := p.lex.Text()
	if len(text) < 3 {
		panic("bad rune literal: " + text)
	}
	r, _, _, err := strconv.UnquoteChar(text[1:], '\'')
	if err != nil {
		// The lexer may allow bad rune literals (>0x0010FFFF and
		// surrogate halves—whatever they are).
		panic(&MalformedLiteral{
			Type:  "rune literal",
			Text:  p.text(),
			Start: p.lex.Start,
			End:   p.lex.End,
		})
	}
	l := &RuneLiteral{Value: r, span: p.span()}
	p.next()
	return l
}
