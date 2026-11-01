package klaracase

// Error codes that the SDK itself sets. The codes of the API are listed with them in errors.go.
const (
	// CodeClientValidation marks a call that the client refused before it sent a request.
	CodeClientValidation = "client_validation"
	// CodeInvalidResponse marks a success answer whose body does not fit the SDK types.
	CodeInvalidResponse = "invalid_response"
)
