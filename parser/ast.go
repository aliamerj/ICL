package parser

import (
	"github.com/aliamerj/icl/tokens"
)

// ---- Position tracking ----

type pos struct {
	Line, Offset int
}

type rangePos struct {
	Start, End pos
}

// ---- Base interfaces ----

type node interface {
	Range() rangePos
}

type Statement interface {
	node
	statementNode()
}

type Expression interface {
	node
	expressionNode()
}

// ---- Identifier: keys, bareword types, references (ubuntu.id), names ----

type Identifier struct {
	Name string
	Rng  rangePos
}

func (i Identifier) Range() rangePos { return i.Rng }
func (i Identifier) expressionNode() {}

// ---- Attribute: one `key = value` line ----

type Attribute struct {
	Name  *Identifier
	Value Expression
	Rng   rangePos
}

func (a Attribute) Range() rangePos { return a.Rng }
func (a Attribute) statementNode()  {}

// Block: the ONE shape for provider, resource, lookup, input, output, module, and nested blocks like filter/route

type Block struct {
	Keyword tokens.Type   // "provider", "resource", "lookup", "filter"
	Labels  []*Identifier // e.g. [aws_instance] — positional, pre-body identifiers
	Name    *Identifier   // the `as app_server` alias — nil if absent
	Body    *Body
	Rng     rangePos
}

func (b Block) Range() rangePos { return b.Rng }
func (b Block) statementNode()  {}

// ---- Body: ordered statements inside { } — order matters for the formatter ----

type Body struct {
	Statements []Statement
	Rng        rangePos
}

func (b Body) Range() rangePos { return b.Rng }

// ---- Program: the file root ----

type Program struct {
	Statements []Statement
	Rng        rangePos
}

func (p Program) Range() rangePos { return p.Rng }

// ---- StringLiteral: "eu-west-1", "hashicorp/aws" ----

type StringLiteral struct {
	Value string // the unescaped string content
	Rng   rangePos
}

func (s StringLiteral) Range() rangePos { return s.Rng }
func (s StringLiteral) expressionNode() {}

// ---- IntLiteral: 123----
type IntLiteral struct {
	Value int64
	Rng   rangePos
}

func (n IntLiteral) Range() rangePos { return n.Rng }
func (n IntLiteral) expressionNode() {}

// ---- FloatLiteral: 14.13 ----
type FloatLiteral struct {
	Value float64
	Rng   rangePos
}

func (n FloatLiteral) Range() rangePos { return n.Rng }
func (n FloatLiteral) expressionNode() {}

// ---- BoolLiteral:  TRUE or FALSE ----
type BoolLiteral struct {
	Value bool
	Rng   rangePos
}

func (b BoolLiteral) Range() rangePos { return b.Rng }
func (b BoolLiteral) expressionNode() {}

type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
	Rng      rangePos
}

func (b BinaryExpr) Range() rangePos { return b.Rng }
func (b BinaryExpr) expressionNode() {}

type ListExpr struct {
	Elements []Expression
	Rng      rangePos
}

func (l *ListExpr) Range() rangePos { return l.Rng }
func (l *ListExpr) expressionNode() {}

type ObjectExpr struct {
	Fields []*Attribute
	Rng    rangePos
}

func (o *ObjectExpr) Range() rangePos { return o.Rng }
func (o *ObjectExpr) expressionNode() {}

type MemberExpr struct {
	Object   Expression
	Property string
	Rng      rangePos
}

func (m *MemberExpr) Range() rangePos { return m.Rng }
func (m *MemberExpr) expressionNode() {}
