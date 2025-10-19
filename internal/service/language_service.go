package service

// LanguageService handles language validation and management
type LanguageService struct {
	supportedLanguages []string
}

// NewLanguageService creates a new language service
func NewLanguageService() *LanguageService {
	return &LanguageService{
		supportedLanguages: []string{"en", "hi", "mr", "gu"},
	}
}

// IsValidLanguage checks if the language code is supported
func (l *LanguageService) IsValidLanguage(lang string) bool {
	for _, supportedLang := range l.supportedLanguages {
		if supportedLang == lang {
			return true
		}
	}
	return false
}

// GetTargetLanguages returns the target languages for translation
// Excludes the source language from the list
func (l *LanguageService) GetTargetLanguages(sourceLang string) []string {
	var targetLanguages []string
	for _, lang := range l.supportedLanguages {
		if lang != sourceLang {
			targetLanguages = append(targetLanguages, lang)
		}
	}
	return targetLanguages
}

// GetSupportedLanguages returns all supported language codes
func (l *LanguageService) GetSupportedLanguages() []string {
	return l.supportedLanguages
}
