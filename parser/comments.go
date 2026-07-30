package parser

import "github.com/aliamerj/icl/lexer"

// AttachComments reattaches scanner comments to nearby block and attribute
// nodes by line adjacency. This deliberately only covers statements, so
// comments inside nested object or list literals are still dropped for now.
func AttachComments(prog *Program, comments []lexer.Comment) {
	if prog == nil || len(comments) == 0 {
		return
	}

	byLine := make(map[int]string, len(comments))
	for _, c := range comments {
		byLine[c.Line] = c.Text
	}

	attachToStatements(prog.Statements, byLine)
}

func attachToStatements(stmts []Statement, byLine map[int]string) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *Block:
			if text, ok := byLine[s.Rng.Start.Line-1]; ok {
				s.LeadingComments = append(s.LeadingComments, text)
			}
			if text, ok := byLine[s.Rng.End.Line]; ok {
				s.TrailingComment = text
			}
			if s.Body != nil {
				attachToStatements(s.Body.Statements, byLine)
			}
		case *Attribute:
			if text, ok := byLine[s.Rng.Start.Line-1]; ok {
				s.LeadingComments = append(s.LeadingComments, text)
			}
			if text, ok := byLine[s.Rng.End.Line]; ok {
				s.TrailingComment = text
			}
		}
	}
}
