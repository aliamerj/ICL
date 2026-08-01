package tokens

//go:generate stringer -type=Type
type Type int

const (
	// Single-character tokens.
	LEFT_PAREN    Type = iota // (
	RIGHT_PAREN               // )
	LEFT_BRACE                // {
	RIGHT_BRACE               // }
	LEFT_BRACKET              // [
	RIGHT_BRACKET             // ]
	DOT                       // .
	MINUS                     // -
	PLUS                      // +
	SEMICOLON                 // ;
	SLASH                     // /
	STAR                      // *
	COMMA                     // ,

	// One or two character tokens.
	BANG          // !
	BANG_EQUAL    // !=
	EQUAL         // =
	EQUAL_EQUAL   // ==
	GREATER       // >
	GREATER_EQUAL // >=
	LESS          // <
	LESS_EQUAL    // <=

	// Literals.
	IDENTIFIER
	STRING
	NUMBER_INT
	NUMBER_FLOAT
	TRUE
	FALSE

	// Keywords
	PROVIDER
  RESOURCE
  AS
  LOOKUP

	EOF
)
