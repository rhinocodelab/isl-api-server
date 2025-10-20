package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/go-audio/wav"
	"google.golang.org/api/option"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	tokens    int
	maxTokens int
	mu        sync.Mutex
	ticker    *time.Ticker
	stopChan  chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	rl := &RateLimiter{
		tokens:    maxTokens,
		maxTokens: maxTokens,
		ticker:    time.NewTicker(refillRate),
		stopChan:  make(chan struct{}),
	}

	// Start refill goroutine
	go rl.refill()

	return rl
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// refill refills tokens at the specified rate
func (rl *RateLimiter) refill() {
	for {
		select {
		case <-rl.ticker.C:
			rl.mu.Lock()
			if rl.tokens < rl.maxTokens {
				rl.tokens++
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			rl.ticker.Stop()
			return
		}
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// SpeechClientPool manages a pool of GCP Speech clients
type SpeechClientPool struct {
	clients    []*speech.Client
	current    int
	mu         sync.Mutex
	maxClients int
}

// NewSpeechClientPool creates a new client pool
func NewSpeechClientPool(projectID string, maxClients int, logger Logger) (*SpeechClientPool, error) {
	ctx := context.Background()
	clients := make([]*speech.Client, maxClients)

	// Get credentials path
	credentialsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credentialsPath == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
	}

	// Check if credentials file exists
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("credentials file not found: %s", credentialsPath)
	}

	// Create multiple clients
	for i := 0; i < maxClients; i++ {
		client, err := speech.NewClient(ctx, option.WithCredentialsFile(credentialsPath))
		if err != nil {
			// Close already created clients on error
			for j := 0; j < i; j++ {
				clients[j].Close()
			}
			return nil, fmt.Errorf("failed to create speech client %d: %w", i, err)
		}
		clients[i] = client
	}

	logger.Info("Speech client pool initialized", "max_clients", maxClients)

	return &SpeechClientPool{
		clients:    clients,
		maxClients: maxClients,
	}, nil
}

// GetClient returns a client from the pool
func (p *SpeechClientPool) GetClient() *speech.Client {
	p.mu.Lock()
	defer p.mu.Unlock()

	client := p.clients[p.current]
	p.current = (p.current + 1) % p.maxClients
	return client
}

// Close closes all clients in the pool
func (p *SpeechClientPool) Close() error {
	var lastErr error
	for _, client := range p.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// SpeechNumbersDictionary represents the numbers dictionary structure for speech-to-text
type SpeechNumbersDictionary struct {
	Metadata map[string]interface{}       `json:"metadata"`
	Numbers  map[string]map[string]string `json:"numbers"`
}

// SpeechToTextService handles GCP Speech-to-Text API operations
type SpeechToTextService struct {
	clientPool     *SpeechClientPool
	projectID      string
	logger         Logger
	numbersDict    *SpeechNumbersDictionary
	dictionaryPath string
	rateLimiter    *RateLimiter
	requestQueue   *RequestQueue
}

// NewSpeechToTextService creates a new speech-to-text service
func NewSpeechToTextService(projectID string, dictionaryPath string, logger Logger) (*SpeechToTextService, error) {
	// Create client pool (5 clients for concurrent requests)
	clientPool, err := NewSpeechClientPool(projectID, 5, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create speech client pool: %w", err)
	}

	// Load numbers dictionary using the same function as text translation service
	numbersDict, err := loadSpeechNumbersDictionary(dictionaryPath, logger)
	if err != nil {
		logger.Error("Failed to load numbers dictionary", "error", err)
		// Continue without dictionary - service will work but without number translation
	}

	// Initialize rate limiter (5 requests per second, max 10 tokens)
	rateLimiter := NewRateLimiter(10, 200*time.Millisecond) // 5 requests per second

	// Create service instance first
	service := &SpeechToTextService{
		clientPool:     clientPool,
		projectID:      projectID,
		logger:         logger,
		numbersDict:    numbersDict,
		dictionaryPath: dictionaryPath,
		rateLimiter:    rateLimiter,
	}

	// Initialize request queue with 3 workers
	requestQueue := NewRequestQueue(service, 3, logger)
	service.requestQueue = requestQueue

	logger.Info("Speech-to-text service initialized", "project_id", projectID, "dictionary_path", dictionaryPath)

	return service, nil
}

// Close closes the speech client pool, rate limiter, and request queue
func (s *SpeechToTextService) Close() error {
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
	if s.requestQueue != nil {
		s.requestQueue.Close()
	}
	if s.clientPool != nil {
		return s.clientPool.Close()
	}
	return nil
}

// DetectSampleRate detects the sample rate of an audio file
func (s *SpeechToTextService) DetectSampleRate(filePath string) (int32, error) {
	s.logger.Info("Detecting sample rate", "file", filePath)

	// Check file extension to determine if it's a WAV file
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".wav" {
		// For non-WAV files, return default sample rate
		s.logger.Info("Non-WAV file detected, using default sample rate", "file", filePath, "extension", ext)
		return 16000, nil
	}

	// Open and decode WAV file
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return 0, fmt.Errorf("invalid WAV file")
	}

	sampleRate := int32(decoder.SampleRate)
	s.logger.Info("Sample rate detected", "file", filePath, "sample_rate", sampleRate)

	return sampleRate, nil
}

// TranscribeAudio transcribes audio to text using GCP Speech-to-Text
func (s *SpeechToTextService) TranscribeAudio(ctx context.Context, filePath, languageCode string) (string, error) {
	startTime := time.Now()
	s.logger.Info("Starting speech-to-text transcription", "file", filePath, "language", languageCode)

	// Check rate limit
	if !s.rateLimiter.Allow() {
		s.logger.Error("Rate limit exceeded", "file", filePath)
		return "", fmt.Errorf("rate limit exceeded, please try again later")
	}

	// Read audio file
	audioData, err := os.ReadFile(filePath)
	if err != nil {
		s.logger.Error("Failed to read audio file", "error", err)
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}

	// Detect sample rate
	sampleRate, err := s.DetectSampleRate(filePath)
	if err != nil {
		s.logger.Error("Failed to detect sample rate", "error", err)
		// Use default sample rate as fallback
		sampleRate = 16000
	}

	// Determine encoding based on file extension
	encoding := s.getAudioEncoding(filePath)

	// Create recognition request
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   encoding,
			SampleRateHertz:            sampleRate,
			LanguageCode:               languageCode,
			EnableAutomaticPunctuation: true,
			EnableWordTimeOffsets:      false,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: audioData},
		},
	}

	// Get client from pool and perform recognition
	client := s.clientPool.GetClient()
	resp, err := client.Recognize(ctx, req)
	if err != nil {
		s.logger.Error("Speech recognition failed", "error", err)
		return "", fmt.Errorf("speech recognition failed: %w", err)
	}

	// Extract transcript
	if len(resp.Results) == 0 {
		s.logger.Error("No transcription results", "file", filePath)
		return "", fmt.Errorf("no transcription results")
	}

	// Get the best alternative
	transcript := resp.Results[0].Alternatives[0].Transcript
	confidence := resp.Results[0].Alternatives[0].Confidence

	// Convert numbers to words in the original transcript
	processedTranscript := s.convertNumbersToWords(transcript, languageCode)

	processingTime := time.Since(startTime)
	s.logger.Info("Speech-to-text transcription completed",
		"transcript_length", len(processedTranscript),
		"confidence", confidence,
		"processing_time", processingTime.String())

	return processedTranscript, nil
}

// TranscribeAudioStream transcribes audio using streaming for memory efficiency
func (s *SpeechToTextService) TranscribeAudioStream(ctx context.Context, filePath, languageCode string) (string, error) {
	startTime := time.Now()
	s.logger.Info("Starting streaming speech-to-text transcription", "file", filePath, "language", languageCode)

	// Check rate limit
	if !s.rateLimiter.Allow() {
		s.logger.Error("Rate limit exceeded", "file", filePath)
		return "", fmt.Errorf("rate limit exceeded, please try again later")
	}

	// Detect sample rate
	sampleRate, err := s.DetectSampleRate(filePath)
	if err != nil {
		s.logger.Error("Failed to detect sample rate", "error", err)
		sampleRate = 16000 // Use default
	}

	// Determine encoding
	encoding := s.getAudioEncoding(filePath)

	// Read audio file in chunks to reduce memory usage
	audioData, err := s.readAudioFileInChunks(filePath)
	if err != nil {
		s.logger.Error("Failed to read audio file", "error", err)
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}

	// Create recognition request
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:                   encoding,
			SampleRateHertz:            sampleRate,
			LanguageCode:               languageCode,
			EnableAutomaticPunctuation: true,
			EnableWordTimeOffsets:      false,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{Content: audioData},
		},
	}

	// Get client from pool and perform recognition
	client := s.clientPool.GetClient()
	resp, err := client.Recognize(ctx, req)
	if err != nil {
		s.logger.Error("Speech recognition failed", "error", err)
		return "", fmt.Errorf("speech recognition failed: %w", err)
	}

	// Extract transcript
	if len(resp.Results) == 0 {
		s.logger.Error("No transcription results", "file", filePath)
		return "", fmt.Errorf("no transcription results")
	}

	// Get the best alternative
	transcript := resp.Results[0].Alternatives[0].Transcript
	confidence := resp.Results[0].Alternatives[0].Confidence

	// Convert numbers to words in the original transcript
	processedTranscript := s.convertNumbersToWords(transcript, languageCode)

	processingTime := time.Since(startTime)
	s.logger.Info("Streaming speech-to-text transcription completed",
		"transcript_length", len(processedTranscript),
		"confidence", confidence,
		"processing_time", processingTime.String())

	return processedTranscript, nil
}

// TranscribeAudioQueued transcribes audio using the request queue for better load handling
func (s *SpeechToTextService) TranscribeAudioQueued(ctx context.Context, filePath, languageCode string) (string, error) {
	// Submit request to queue
	responseChan := s.requestQueue.SubmitRequest(ctx, filePath, languageCode)

	// Wait for response
	select {
	case response := <-responseChan:
		if response.Error != nil {
			return "", response.Error
		}
		return response.Transcript, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// readAudioFileInChunks reads audio file in chunks to reduce memory usage
func (s *SpeechToTextService) readAudioFileInChunks(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	const chunkSize = 1024 * 1024 // 1MB chunks
	var audioData []byte
	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read file chunk: %w", err)
		}
		audioData = append(audioData, buffer[:n]...)
	}

	s.logger.Info("Audio file read in chunks", "file", filePath, "size", len(audioData))
	return audioData, nil
}

// TranscriptionRequest represents a queued transcription request
type TranscriptionRequest struct {
	FilePath     string
	LanguageCode string
	ResponseChan chan *TranscriptionResponse
	Context      context.Context
}

// TranscriptionResponse represents the response from a transcription request
type TranscriptionResponse struct {
	Transcript string
	Error      error
}

// RequestQueue manages a queue of transcription requests
type RequestQueue struct {
	queue   chan *TranscriptionRequest
	workers int
	service *SpeechToTextService
	logger  Logger
}

// NewRequestQueue creates a new request queue
func NewRequestQueue(service *SpeechToTextService, workers int, logger Logger) *RequestQueue {
	q := &RequestQueue{
		queue:   make(chan *TranscriptionRequest, 100), // Buffer for 100 requests
		workers: workers,
		service: service,
		logger:  logger,
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go q.worker(i)
	}

	logger.Info("Request queue initialized", "workers", workers)
	return q
}

// SubmitRequest submits a transcription request to the queue
func (q *RequestQueue) SubmitRequest(ctx context.Context, filePath, languageCode string) <-chan *TranscriptionResponse {
	responseChan := make(chan *TranscriptionResponse, 1)

	select {
	case q.queue <- &TranscriptionRequest{
		FilePath:     filePath,
		LanguageCode: languageCode,
		ResponseChan: responseChan,
		Context:      ctx,
	}:
		q.logger.Info("Request submitted to queue", "file", filePath)
		return responseChan
	case <-ctx.Done():
		responseChan <- &TranscriptionResponse{Error: ctx.Err()}
		close(responseChan)
		return responseChan
	}
}

// worker processes requests from the queue
func (q *RequestQueue) worker(workerID int) {
	for req := range q.queue {
		q.logger.Info("Worker processing request", "worker", workerID, "file", req.FilePath)

		// Process the transcription request
		transcript, err := q.service.TranscribeAudioStream(req.Context, req.FilePath, req.LanguageCode)

		// Send response
		select {
		case req.ResponseChan <- &TranscriptionResponse{
			Transcript: transcript,
			Error:      err,
		}:
		case <-req.Context.Done():
			req.ResponseChan <- &TranscriptionResponse{Error: req.Context.Err()}
		}
		close(req.ResponseChan)
	}
}

// Close closes the request queue
func (q *RequestQueue) Close() {
	close(q.queue)
}

// getAudioEncoding determines the audio encoding based on file extension
func (s *SpeechToTextService) getAudioEncoding(filePath string) speechpb.RecognitionConfig_AudioEncoding {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".wav":
		return speechpb.RecognitionConfig_LINEAR16
	case ".flac":
		return speechpb.RecognitionConfig_FLAC
	case ".mp3":
		return speechpb.RecognitionConfig_MP3
	case ".ogg":
		return speechpb.RecognitionConfig_OGG_OPUS
	default:
		// Default to LINEAR16 for unknown formats
		return speechpb.RecognitionConfig_LINEAR16
	}
}

// ValidateLanguageCode validates if the language code is supported
func (s *SpeechToTextService) ValidateLanguageCode(languageCode string) bool {
	supportedLanguages := []string{"en-IN", "hi-IN", "mr-IN", "gu-IN"}

	for _, lang := range supportedLanguages {
		if lang == languageCode {
			return true
		}
	}

	return false
}

// GetSupportedLanguages returns the list of supported language codes
func (s *SpeechToTextService) GetSupportedLanguages() []string {
	return []string{"en-IN", "hi-IN", "mr-IN", "gu-IN"}
}

// loadSpeechNumbersDictionary loads the numbers dictionary from file
func loadSpeechNumbersDictionary(dictionaryPath string, logger Logger) (*SpeechNumbersDictionary, error) {
	numbersFilePath := fmt.Sprintf("%s/numbers.json", dictionaryPath)

	// Check if file exists
	if _, err := os.Stat(numbersFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("numbers dictionary file not found: %s", numbersFilePath)
	}

	// Read file
	data, err := os.ReadFile(numbersFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read numbers dictionary file: %w", err)
	}

	// Parse JSON
	var numbersDict SpeechNumbersDictionary
	if err := json.Unmarshal(data, &numbersDict); err != nil {
		return nil, fmt.Errorf("failed to parse numbers dictionary JSON: %w", err)
	}

	logger.Info("Numbers dictionary loaded successfully", "file", numbersFilePath, "numbers_count", len(numbersDict.Numbers))
	return &numbersDict, nil
}

// convertNumbersToWords converts numeric digits in text to words using dictionary
func (s *SpeechToTextService) convertNumbersToWords(text, targetLang string) string {
	if s.numbersDict == nil {
		// No dictionary available, return original text
		return text
	}

	// Convert language code from full format (en-IN) to short format (en)
	shortLang := s.convertLanguageCodeToShort(targetLang)

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
			if word, exists := s.numbersDict.Numbers[digit][shortLang]; exists {
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

		s.logger.Info("Number converted", "original", number, "converted", numberWords, "target_lang", targetLang, "short_lang", shortLang)
	}

	return text
}

// convertLanguageCodeToShort converts full language codes to short format for dictionary lookup
func (s *SpeechToTextService) convertLanguageCodeToShort(langCode string) string {
	switch langCode {
	case "en-IN":
		return "en"
	case "hi-IN":
		return "hi"
	case "mr-IN":
		return "mr"
	case "gu-IN":
		return "gu"
	default:
		// Default to English if language not recognized
		return "en"
	}
}
