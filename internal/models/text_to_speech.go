package models

// TextToSpeechRequest represents the request payload for text-to-speech
type TextToSpeechRequest struct {
	Texts []string `json:"texts" binding:"required"`
}

// AudioFileResponse represents a single text with its audio files
type AudioFileResponse struct {
	Text  string            `json:"text"`
	Index int               `json:"index"`
	Files map[string]string `json:"files"`
}

// TextToSpeechResponse represents the response payload for text-to-speech
type TextToSpeechResponse struct {
	RequestID       string                 `json:"request_id"`
	FolderPath      string                 `json:"folder_path"`
	TotalTexts      int                    `json:"total_texts"`
	TotalAudioFiles int                    `json:"total_audio_files"`
	Languages       []string               `json:"languages"`
	AudioFiles      []AudioFileResponse    `json:"audio_files"`
	ProcessingTime  string                 `json:"processing_time"`
	Status          string                 `json:"status"`
	Errors          []AudioGenerationError `json:"errors,omitempty"`
}

// AudioGenerationError represents an error in audio generation
type AudioGenerationError struct {
	TextIndex int    `json:"text_index"`
	Language  string `json:"language"`
	Error     string `json:"error"`
}

// TextToSpeechError represents an error response for text-to-speech
type TextToSpeechError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
