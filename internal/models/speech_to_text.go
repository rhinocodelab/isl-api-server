package models

// SpeechToTextResponse represents the response for speech-to-text conversion
type SpeechToTextResponse struct {
	OriginalTranscript string            `json:"original_transcript"`
	OriginalLanguage   string            `json:"original_language"`
	SampleRate         int32             `json:"sample_rate"`
	AudioFormat        string            `json:"audio_format"`
	FileSize           int64             `json:"file_size"`
	Translations       map[string]string `json:"translations"`
	ProcessingTime     string            `json:"processing_time,omitempty"`
}

// SpeechToTextError represents an error response for speech-to-text conversion
type SpeechToTextError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
