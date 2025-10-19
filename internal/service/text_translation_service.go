package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	translate "cloud.google.com/go/translate/apiv3"
	"cloud.google.com/go/translate/apiv3/translatepb"
	"google.golang.org/api/option"
)

// NumbersDictionary represents the numbers dictionary structure
type NumbersDictionary struct {
	Metadata map[string]interface{}       `json:"metadata"`
	Numbers  map[string]map[string]string `json:"numbers"`
}

// TranslationService handles GCP Translation API operations
type TranslationService struct {
	client         *translate.TranslationClient
	projectID      string
	logger         Logger
	numbersDict    *NumbersDictionary
	dictionaryPath string
}

// Logger interface for logging
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// NewTranslationService creates a new translation service
func NewTranslationService(projectID string, dictionaryPath string, logger Logger) (*TranslationService, error) {
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

	// Load numbers dictionary
	numbersDict, err := loadNumbersDictionary(dictionaryPath, logger)
	if err != nil {
		logger.Error("Failed to load numbers dictionary", "error", err)
		// Continue without dictionary - service will work but without number translation
	}

	logger.Info("Translation service initialized", "project_id", projectID, "dictionary_path", dictionaryPath)

	return &TranslationService{
		client:         client,
		projectID:      projectID,
		logger:         logger,
		numbersDict:    numbersDict,
		dictionaryPath: dictionaryPath,
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

			// Convert numbers to words before translation
			processedText := t.convertNumbersToWords(text, lang)

			translatedText, err := t.TranslateText(translationCtx, processedText, sourceLang, lang)

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

// loadNumbersDictionary loads the numbers dictionary from file
func loadNumbersDictionary(dictionaryPath string, logger Logger) (*NumbersDictionary, error) {
	numbersFilePath := fmt.Sprintf("%s/numbers.json", dictionaryPath)

	// Check if file exists
	if _, err := os.Stat(numbersFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("numbers dictionary file not found: %s", numbersFilePath)
	}

	// Read file
	data, err := ioutil.ReadFile(numbersFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read numbers dictionary file: %w", err)
	}

	// Parse JSON
	var numbersDict NumbersDictionary
	if err := json.Unmarshal(data, &numbersDict); err != nil {
		return nil, fmt.Errorf("failed to parse numbers dictionary JSON: %w", err)
	}

	logger.Info("Numbers dictionary loaded successfully", "file", numbersFilePath, "numbers_count", len(numbersDict.Numbers))
	return &numbersDict, nil
}

// convertNumbersToWords converts numeric digits in text to words using dictionary
func (t *TranslationService) convertNumbersToWords(text, targetLang string) string {
	if t.numbersDict == nil {
		// No dictionary available, return original text
		return text
	}

	// Find all numbers in text
	numberRegex := regexp.MustCompile(`\b\d+\b`)
	numbers := numberRegex.FindAllString(text, -1)

	// Convert each number
	for _, number := range numbers {
		// Split number into individual digits
		digits := strings.Split(number, "")
		var digitWords []string

		// Convert each digit to word
		for _, digit := range digits {
			if word, exists := t.numbersDict.Numbers[digit][targetLang]; exists {
				digitWords = append(digitWords, word)
			} else {
				// If digit not found in dictionary, keep original digit
				digitWords = append(digitWords, digit)
			}
		}

		// Join digit words with spaces
		numberWords := strings.Join(digitWords, " ")

		// Replace number in text
		text = strings.Replace(text, number, numberWords, -1)

		t.logger.Info("Number converted", "original", number, "converted", numberWords, "target_lang", targetLang)
	}

	return text
}
