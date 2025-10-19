package models

// AudioLanguageDetectionResponse represents the response for audio language detection
type AudioLanguageDetectionResponse struct {
	DetectedLanguage string `json:"detected_language"`
	LanguageCode     string `json:"language_code"`
	Confidence       string `json:"confidence,omitempty"`
	AudioFormat      string `json:"audio_format"`
	FileSize         int64  `json:"file_size"`
	ProcessingTime   string `json:"processing_time,omitempty"`
}

// AudioLanguageDetectionError represents an error response for audio language detection
type AudioLanguageDetectionError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
