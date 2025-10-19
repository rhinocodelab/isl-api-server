package models

// TranslationRequest represents the request payload for text translation
type TranslationRequest struct {
	Text           string `json:"text" binding:"required"`
	SourceLanguage string `json:"source_language" binding:"required"`
}

// TranslationResponse represents the response payload for text translation
type TranslationResponse struct {
	OriginalText   string            `json:"original_text"`
	SourceLanguage string            `json:"source_language"`
	Translations   map[string]string `json:"translations"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
