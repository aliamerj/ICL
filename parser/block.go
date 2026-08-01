package parser

import (
	"fmt"

	"github.com/aliamerj/icl/diagnostics"
	"github.com/aliamerj/icl/tokens"
)

// parseBlock is the one grammar function for every "KEYWORD label [as name] { body }"
// construct in the language.
func (p *parser) parseBlock() *Block {
	kwTok := p.advance()

	labelTok, ok := p.expect(tokens.IDENTIFIER)
	if !ok {
		return nil
	}

	blk := &Block{
		Keyword: kwTok.Type,
		Labels:  []*Identifier{{Name: labelTok.Lexeme, Rng: rangeOf(labelTok)}},
	}

	switch {
	case p.cur().Type == tokens.AS:
		p.advance()

		nameTok, ok := p.expect(tokens.IDENTIFIER)
		if !ok {
			return nil
		}

		blk.Name = &Identifier{
			Name: nameTok.Lexeme,
			Rng:  rangeOf(nameTok),
		}

	case kwTok.Type == tokens.RESOURCE, kwTok.Type == tokens.LOOKUP:
		p.reporter.ErrorAtOffsetWithCode(
			p.cur().Offset,
			diagnostics.MISSING_REQUIRED_NAME,
			fmt.Sprintf("%s %q must have a name", kwTok.Type, labelTok.Lexeme),
			fmt.Sprintf("add `as <name>`, e.g. `%s %s as my_%s`",
				kwTok.Type.String(),
				labelTok.Lexeme,
				kwTok.Type.String(),
			),
		)
		return nil
	}

	if _, ok := p.expect(tokens.LEFT_BRACE); !ok {
		return nil
	}

	body := p.parseBody()
	if body == nil {
		return nil
	}
	blk.Body = body

	endTok, ok := p.expect(tokens.RIGHT_BRACE)
	if !ok {
		return nil
	}

	blk.Rng = spanOf(kwTok, endTok)

	return blk
}
