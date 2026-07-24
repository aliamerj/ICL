package parser

// ---- Position tracking ----

type Pos struct {
	Line, Offset int
}

type Range struct {
	Start, End Pos
}

// ---- Base interfaces ----

type Node interface {
	Range() Range
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

// ---- Identifier: keys, bareword types, references (ubuntu.id), names ----

type Identifier struct {
	Name string
	Rng  Range
}

func (i *Identifier) Range() Range    { return i.Rng }
func (i *Identifier) expressionNode() {}

// ---- Attribute: one `key = value` line ----

type Attribute struct {
	Name  *Identifier
	Value Expression
	Rng   Range
}

func (a *Attribute) Range() Range   { return a.Rng }
func (a *Attribute) statementNode() {}

// Block: the ONE shape for provider, resource, lookup, input, output, module, and nested blocks like filter/route

type Block struct {
	Keyword string        // "provider", "resource", "lookup", "filter"
	Labels  []*Identifier // e.g. [aws_instance] — positional, pre-body identifiers
	Name    *Identifier   // the `as app_server` alias — nil if absent
	Body    *Body
	Rng     Range
}

func (b *Block) Range() Range   { return b.Rng }
func (b *Block) statementNode() {}

// ---- Body: ordered statements inside { } — order matters for the formatter ----

type Body struct {
	Statements []Statement
	Rng        Range
}

func (b *Body) Range() Range { return b.Rng }

// ---- Program: the file root ----

type Program struct {
	Statements []Statement
	Rng        Range
}

func (p *Program) Range() Range { return p.Rng }

// ---- StringLiteral: "eu-west-1", "hashicorp/aws" ----

type StringLiteral struct {
	Value string // the unescaped string content
	Rng   Range
}

func (s *StringLiteral) Range() Range    { return s.Rng }
func (s *StringLiteral) expressionNode() {}
