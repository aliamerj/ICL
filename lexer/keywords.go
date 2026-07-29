package lexer

import "github.com/aliamerj/icl/tokens"

var keywords = map[string]tokens.Type{
	"provider": tokens.PROVIDER,
	"resource": tokens.RESOURCE,
	"as":       tokens.AS,
}
