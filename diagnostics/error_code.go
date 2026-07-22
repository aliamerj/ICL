package diagnostics

type errorCode string

const (
	UNEXPECTED_CHAR             errorCode = "E0001"
	UNTERMINATED_STRING_LITERAL errorCode = "E0002"
	ERROR_NUMBER_LITERAL        errorCode = "E0003"
)
