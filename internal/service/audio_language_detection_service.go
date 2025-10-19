package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/genai"
)

// AudioLanguageDetectionService handles audio language detection using Google Gemini
type AudioLanguageDetectionService struct {
	geminiClient *genai.Client
	logger       Logger
}

// NewAudioLanguageDetectionService creates a new audio language detection service
func NewAudioLanguageDetectionService(geminiAPIKey string, logger Logger) (*AudioLanguageDetectionService, error) {
	if geminiAPIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	// Create client with API key and BackendGeminiAPI
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  geminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	logger.Info("Audio language detection service initialized")

	return &AudioLanguageDetectionService{
		geminiClient: client,
		logger:       logger,
	}, nil
}

// Close closes the Gemini client
func (a *AudioLanguageDetectionService) Close() error {
	// Gemini client doesn't have a Close method, just return nil
	return nil
}

// DetectLanguage detects the language from an audio file
func (a *AudioLanguageDetectionService) DetectLanguage(ctx context.Context, audioFilePath string) (string, error) {
	startTime := time.Now()
	a.logger.Info("Starting audio language detection", "file", audioFilePath)

	// Read the audio file
	audioData, err := os.ReadFile(audioFilePath)
	if err != nil {
		a.logger.Error("Failed to read audio file", "error", err)
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}

	// Determine MIME type based on file extension
	fileExt := strings.ToLower(filepath.Ext(audioFilePath))
	mimeType := "audio/aac" // default
	switch fileExt {
	case ".wav":
		mimeType = "audio/wav"
	case ".mp3":
		mimeType = "audio/mp3"
	case ".aiff":
		mimeType = "audio/aiff"
	case ".aac":
		mimeType = "audio/aac"
	case ".ogg":
		mimeType = "audio/ogg"
	case ".flac":
		mimeType = "audio/flac"
	}

	// Create prompt for language detection
	parts := []*genai.Part{
		genai.NewPartFromText("Detect the language spoken in this audio file. Return only one of these valid languages: Marathi, English, Hindi, Gujarati. Do not include any additional text or explanation."),
		&genai.Part{
			InlineData: &genai.Blob{
				MIMEType: mimeType,
				Data:     audioData,
			},
		},
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	// Generate content using Gemini
	result, err := a.geminiClient.Models.GenerateContent(ctx, "gemini-2.5-flash", contents, nil)
	if err != nil {
		a.logger.Error("Failed to generate content with Gemini", "error", err)
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	// Extract detected language
	detectedLanguage := strings.TrimSpace(result.Text())
	processingTime := time.Since(startTime)

	// Map language to language code
	languageCode := a.GetLanguageCode(detectedLanguage)

	a.logger.Info("Audio language detection completed",
		"detected_language", detectedLanguage,
		"language_code", languageCode,
		"processing_time", processingTime.String())

	return detectedLanguage, nil
}

// ValidateAudioFile validates the uploaded audio file
func (a *AudioLanguageDetectionService) ValidateAudioFile(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", filePath)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Check file size (max 10MB)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if fileInfo.Size() > maxSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)", fileInfo.Size(), maxSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	supportedExtensions := []string{".wav", ".mp3", ".aiff", ".aac", ".ogg", ".flac"}

	for _, supportedExt := range supportedExtensions {
		if ext == supportedExt {
			return nil
		}
	}

	return fmt.Errorf("unsupported audio format: %s (supported: wav, mp3, aiff, aac, ogg, flac)", ext)
}

// GetSupportedAudioFormats returns the list of supported audio formats
func (a *AudioLanguageDetectionService) GetSupportedAudioFormats() []string {
	return []string{
		"audio/wav",
		"audio/mp3",
		"audio/aiff",
		"audio/aac",
		"audio/ogg",
		"audio/flac",
	}
}

// GetLanguageCode maps detected language to language code
func (a *AudioLanguageDetectionService) GetLanguageCode(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))

	switch language {
	case "english":
		return "en-IN"
	case "hindi":
		return "hi-IN"
	case "marathi":
		return "mr-IN"
	case "gujarati":
		return "gu-IN"
	default:
		// Default to English if language not recognized
		return "en-IN"
	}
}
