package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	translate "cloud.google.com/go/translate/apiv3"
	"cloud.google.com/go/translate/apiv3/translatepb"
	"google.golang.org/api/option"
)

// TranslationService handles GCP Translation API operations
type TranslationService struct {
	client    *translate.TranslationClient
	projectID string
	logger    Logger
}

// Logger interface for logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// NewTranslationService creates a new translation service
func NewTranslationService(projectID string, logger Logger) (*TranslationService, error) {
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
	client, err := translate.NewTranslationClient(ctx, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create translation client: %w", err)
	}

	logger.Info("Translation service initialized", "project_id", projectID)

	return &TranslationService{
		client:    client,
		projectID: projectID,
		logger:    logger,
	}, nil
}

// Close closes the translation client
func (t *TranslationService) Close() error {
	return t.client.Close()
}

// TranslateText translates text from source language to target language
func (t *TranslationService) TranslateText(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	t.logger.Info("Starting translation", "source", sourceLang, "target", targetLang, "text_length", len(text))

	req := &translatepb.TranslateTextRequest{
		Parent:             fmt.Sprintf("projects/%s/locations/global", t.projectID),
		SourceLanguageCode: sourceLang,
		TargetLanguageCode: targetLang,
		MimeType:           "text/plain",
		Contents:           []string{text},
	}

	resp, err := t.client.TranslateText(ctx, req)
	if err != nil {
		t.logger.Error("Translation failed", "error", err, "source", sourceLang, "target", targetLang)
		return "", fmt.Errorf("translation failed: %w", err)
	}

	if len(resp.GetTranslations()) == 0 {
		t.logger.Error("No translation returned", "source", sourceLang, "target", targetLang)
		return "", fmt.Errorf("no translation returned")
	}

	translatedText := resp.GetTranslations()[0].GetTranslatedText()
	t.logger.Info("Translation completed", "source", sourceLang, "target", targetLang, "translated_length", len(translatedText))

	return translatedText, nil
}

// TranslateToMultipleLanguagesConcurrent translates text to multiple target languages concurrently using goroutines
func (t *TranslationService) TranslateToMultipleLanguagesConcurrent(ctx context.Context, text, sourceLang string, targetLanguages []string) (map[string]string, error) {
	t.logger.Info("Starting concurrent multi-language translation", "source", sourceLang, "targets", targetLanguages, "text_length", len(text))

	var wg sync.WaitGroup
	translations := make(map[string]string)
	errors := make(map[string]error)
	mu := sync.Mutex{}

	// Start concurrent translations for each target language
	for _, targetLang := range targetLanguages {
		wg.Add(1)
		go func(lang string) {
			defer wg.Done()

			// Create individual context for each translation with timeout
			translationCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			translatedText, err := t.TranslateText(translationCtx, text, sourceLang, lang)

			// Thread-safe access to shared data
			mu.Lock()
			if err != nil {
				errors[lang] = err
				t.logger.Error("Translation failed for language", "target", lang, "error", err)
			} else {
				translations[lang] = translatedText
				t.logger.Info("Translation completed", "target", lang, "translated_length", len(translatedText))
			}
			mu.Unlock()
		}(targetLang)
	}

	// Wait for all translations to complete
	wg.Wait()

	// Handle results and errors
	if len(errors) > 0 {
		t.logger.Error("Some translations failed", "failed_count", len(errors), "success_count", len(translations), "errors", errors)

		// Return partial results with error information
		// This allows clients to get available translations even if some fail
		if len(translations) == 0 {
			// If no translations succeeded, return the first error
			for _, err := range errors {
				return nil, fmt.Errorf("all translations failed: %w", err)
			}
		}

		// Log partial success
		t.logger.Info("Partial translation success", "successful", len(translations), "failed", len(errors))
	}

	t.logger.Info("Concurrent multi-language translation completed", "source", sourceLang, "translations_count", len(translations), "errors_count", len(errors))
	return translations, nil
}
