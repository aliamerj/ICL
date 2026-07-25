package diagnostics

type errorCode string

const (
	UNEXPECTED_CHAR             errorCode = "E0001"
	UNTERMINATED_STRING_LITERAL errorCode = "E0002"
	ERROR_NUMBER_LITERAL        errorCode = "E0003"

	// Parser
	UNEXPECTED_TOKEN    errorCode = "E0101"
	EXPECTED_IDENTIFIER errorCode = "E0102"
	EXPECTED_BLOCK_OPEN errorCode = "E0103"
	UNCLOSED_BLOCK      errorCode = "E0104"
	DUPLICATE_NAME      errorCode = "E0105"

	// eval
	INVALID_PROVIDER_BLOCK errorCode = "E0201"
	TYPE_MISMATCH          errorCode = "E0202"
	MISSING_REQUIRED_FIELD errorCode = "E0203"
	UNSUPPORTED_EXPRESSION errorCode = "E0204"
	UNDEFINED_REFERENCE    errorCode = "E0205"
	DIVISION_BY_ZERO       errorCode = "E0206"
)
