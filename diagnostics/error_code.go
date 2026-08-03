package diagnostics

type errorCode string

const (
	UNEXPECTED_CHAR             errorCode = "E0001"
	UNTERMINATED_STRING_LITERAL errorCode = "E0002"
	ERROR_NUMBER_LITERAL        errorCode = "E0003"

	// Parser
	UNEXPECTED_TOKEN      errorCode = "E0101"
	EXPECTED_IDENTIFIER   errorCode = "E0102"
	EXPECTED_BLOCK_OPEN   errorCode = "E0103"
	UNCLOSED_BLOCK        errorCode = "E0104"
	MISSING_REQUIRED_NAME errorCode = "E0108"

	// eval
	INVALID_PROVIDER_BLOCK errorCode = "E0201"
	TYPE_MISMATCH          errorCode = "E0202"
	MISSING_REQUIRED_FIELD errorCode = "E0203"
	UNSUPPORTED_EXPRESSION errorCode = "E0204"
	UNDEFINED_REFERENCE    errorCode = "E0205"
	DIVISION_BY_ZERO       errorCode = "E0206"
	INVALID_TYPE           errorCode = "E0207"
	INVALID_FIELD_ACCESS   errorCode = "E0208"

	INVALID_REFERENCE      errorCode = "E0210"
	INVALID_PROVIDER_REF   errorCode = "E0211"
	UNDEFINED_PROVIDER     errorCode = "E0212"
	UNDEFINED_FIELD        errorCode = "E0213"
	AMBIGUOUS_PROVIDER_REF errorCode = "E0214"
	INVALID_RESOURCE_BLOCK errorCode = "E0215"
	DUPLICATE_NAME         errorCode = "E0216"
	INVALID_INDEX          errorCode = "E0217"
	INDEX_OUT_OF_RANGE     errorCode = "E0218"
	MISSING_OUTPUT_VALUE   errorCode = "E0220"
	UNKNOWN_ATTRIBUTE      errorCode = "E0221"
)
