package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"google.golang.org/api/option"

	"isl-api-server/config"
	"isl-api-server/internal/models"
)

// VoiceConfig represents voice configuration for a language
type VoiceConfig struct {
	LanguageCode string
	Name         string
	SsmlGender   texttospeechpb.SsmlVoiceGender
}

// TextToSpeechService handles GCP Text-to-Speech API operations
type TextToSpeechService struct {
	client    *texttospeech.Client
	projectID string
	config    *config.Config
	logger    Logger
}

// NewTextToSpeechService creates a new text-to-speech service
func NewTextToSpeechService(projectID string, cfg *config.Config, logger Logger) (*TextToSpeechService, error) {
	ctx := context.Background()

	// Get credentials path from environment variable
	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsPath == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
	}

	// Check if credentials file exists
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("credentials file not found: %s", credentialsPath)
	}

	// Create client with explicit credentials
	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS client: %w", err)
	}

	logger.Info("Text-to-speech service initialized", "project_id", projectID, "audio_db_path", cfg.AudioDBPath)

	return &TextToSpeechService{
		client:    client,
		projectID: projectID,
		config:    cfg,
		logger:    logger,
	}, nil
}

// Close closes the TTS client
func (t *TextToSpeechService) Close() error {
	return t.client.Close()
}

// GenerateAudioFiles generates audio files for all texts in all languages
func (t *TextToSpeechService) GenerateAudioFiles(ctx context.Context, requestID string, texts []string) (*models.TextToSpeechResponse, error) {
	startTime := time.Now()
	t.logger.Info("Starting audio generation", "request_id", requestID, "texts_count", len(texts))

	// Get absolute path for audio database
	audioDBPath, err := t.getAbsolutePath(t.config.AudioDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve audio DB path: %w", err)
	}

	// Create request folder
	requestFolder := filepath.Join(audioDBPath, requestID)
	if err := os.MkdirAll(requestFolder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create request folder: %w", err)
	}

	// Generate paths for all audio files
	audioFiles := t.generateAudioFilePaths(requestID, texts)

	// Generate audio files concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []models.AudioGenerationError

	// Voice configurations for each language
	voiceConfigs := map[string]VoiceConfig{
		"en-IN": {
			LanguageCode: "en-IN",
			Name:         "en-IN-Chirp3-HD-Achernar",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},
		"hi-IN": {
			LanguageCode: "hi-IN",
			Name:         "hi-IN-Chirp3-HD-Achernar",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},
		"mr-IN": {
			LanguageCode: "mr-IN",
			Name:         "mr-IN-Chirp3-HD-Achernar",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},
		"gu-IN": {
			LanguageCode: "gu-IN",
			Name:         "gu-IN-Chirp3-HD-Achernar",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},
	}

	// Generate audio for each text in each language
	for i, text := range texts {
		for lang := range voiceConfigs {
			wg.Add(1)
			go func(textIndex int, textContent, language string) {
				defer wg.Done()

				// Create individual context for each generation with timeout
				generationCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()

				err := t.generateAudioFile(generationCtx, textContent, textIndex+1, language, voiceConfigs[language], requestFolder)
				if err != nil {
					mu.Lock()
					errors = append(errors, models.AudioGenerationError{
						TextIndex: textIndex + 1,
						Language:  language,
						Error:     err.Error(),
					})
					mu.Unlock()
					t.logger.Error("Audio generation failed", "text_index", textIndex+1, "language", language, "error", err)
				} else {
					t.logger.Info("Audio generation completed", "text_index", textIndex+1, "language", language)
				}
			}(i, text, lang)
		}
	}

	// Wait for all generations to complete
	wg.Wait()

	processingTime := time.Since(startTime)
	status := "completed"
	if len(errors) > 0 {
		status = "partial_success"
	}

	t.logger.Info("Audio generation completed", "request_id", requestID, "processing_time", processingTime.String(), "errors_count", len(errors))

	return &models.TextToSpeechResponse{
		RequestID:       requestID,
		FolderPath:      requestFolder,
		TotalTexts:      len(texts),
		TotalAudioFiles: len(texts) * 4, // 4 languages
		Languages:       []string{"en-IN", "hi-IN", "mr-IN", "gu-IN"},
		AudioFiles:      audioFiles,
		ProcessingTime:  processingTime.String(),
		Status:          status,
		Errors:          errors,
	}, nil
}

// generateAudioFilePaths generates the file paths for all audio files
func (t *TextToSpeechService) generateAudioFilePaths(requestID string, texts []string) []models.AudioFileResponse {
	var audioFiles []models.AudioFileResponse

	for i, text := range texts {
		files := make(map[string]string)
		for _, lang := range []string{"en-IN", "hi-IN", "mr-IN", "gu-IN"} {
			fileName := fmt.Sprintf("text_%d_%s.wav", i+1, lang)
			filePath := filepath.Join(t.config.AudioDBPath, requestID, fileName)
			files[lang] = filePath
		}

		audioFiles = append(audioFiles, models.AudioFileResponse{
			Text:  text,
			Index: i + 1,
			Files: files,
		})
	}

	return audioFiles
}

// generateAudioFile generates a single audio file
func (t *TextToSpeechService) generateAudioFile(ctx context.Context, text string, textIndex int, language string, voiceConfig VoiceConfig, requestFolder string) error {
	// Perform the text-to-speech request on the text input with the selected
	// voice parameters and audio file type.
	req := texttospeechpb.SynthesizeSpeechRequest{
		// Set the text input to be synthesized.
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		// Build the voice request, select the language code and the SSML voice gender.
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: voiceConfig.LanguageCode,
			Name:         voiceConfig.Name,
			SsmlGender:   voiceConfig.SsmlGender,
		},
		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   texttospeechpb.AudioEncoding_LINEAR16,
			SampleRateHertz: 16000,
			SpeakingRate:    1.0,
			Pitch:           0.0,
			VolumeGainDb:    0.0,
		},
	}

	// Synthesize speech
	resp, err := t.client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		return fmt.Errorf("speech synthesis failed: %w", err)
	}

	// Create file path
	fileName := fmt.Sprintf("text_%d_%s.wav", textIndex, language)
	filePath := filepath.Join(requestFolder, fileName)

	// The resp's AudioContent is binary.
	if err := os.WriteFile(filePath, resp.AudioContent, 0644); err != nil {
		return fmt.Errorf("failed to write audio file: %w", err)
	}

	t.logger.Info("Audio content written to file", "file", filePath, "size", len(resp.AudioContent))
	return nil
}

// getAbsolutePath returns the absolute path for a given path
func (t *TextToSpeechService) getAbsolutePath(relativePath string) (string, error) {
	if filepath.IsAbs(relativePath) {
		return relativePath, nil
	}

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	return filepath.Join(wd, relativePath), nil
}

// ValidateTexts validates the input texts
func (t *TextToSpeechService) ValidateTexts(texts []string) error {
	if len(texts) == 0 {
		return fmt.Errorf("texts array cannot be empty")
	}

	if len(texts) > 10 {
		return fmt.Errorf("maximum 10 texts allowed per request")
	}

	for i, text := range texts {
		if len(strings.TrimSpace(text)) == 0 {
			return fmt.Errorf("text at index %d cannot be empty", i)
		}

		if len(text) > 500 {
			return fmt.Errorf("text at index %d exceeds maximum length of 500 characters", i)
		}
	}

	return nil
}
