# Language Dictionary

This directory contains pre-translated dictionaries for common words and phrases in all supported languages of the ISL API Server.

## 📁 Dictionary Files

### `numbers.json`
Contains number translations from 0-1000 in all four languages:
- **English** (en)
- **Hindi** (hi) - हिन्दी
- **Marathi** (mr) - मराठी
- **Gujarati** (gu) - ગુજરાતી

**Example:**
```json
{
  "1": {
    "en": "one",
    "hi": "एक",
    "mr": "एक",
    "gu": "એક"
  }
}
```

### `common_words.json`
Contains common everyday words and their translations:
- Family members (mother, father, brother, sister)
- Basic emotions (happy, sad, love)
- Common objects (water, food, home)
- Descriptive words (big, small, good, bad)
- Time-related words (day, night, morning, evening)

### `greetings.json`
Contains greetings and polite expressions:
- Time-based greetings (good morning, good evening)
- Polite expressions (thank you, please, excuse me)
- Special occasions (happy birthday, congratulations)
- Farewell expressions (goodbye, see you later)

## 🎯 Usage

These dictionaries can be used for:
- **Offline Translation**: Quick lookups without API calls
- **Caching**: Pre-translated common words for faster responses
- **Fallback**: Backup translations when API is unavailable
- **Testing**: Consistent translations for testing purposes
- **Learning**: Reference for language learning applications

## 📊 Dictionary Structure

Each dictionary file follows this structure:

```json
{
  "metadata": {
    "version": "1.0.0",
    "created": "2025-01-19",
    "description": "Description of the dictionary",
    "languages": ["en", "hi", "mr", "gu"]
  },
  "category": {
    "word_key": {
      "en": "English translation",
      "hi": "Hindi translation",
      "mr": "Marathi translation",
      "gu": "Gujarati translation"
    }
  }
}
```

## 🔧 Integration

### Loading Dictionary in Go
```go
package main

import (
    "encoding/json"
    "io/ioutil"
)

type Dictionary struct {
    Metadata map[string]interface{} `json:"metadata"`
    Words    map[string]map[string]string `json:"words"`
}

func LoadDictionary(filename string) (*Dictionary, error) {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    
    var dict Dictionary
    err = json.Unmarshal(data, &dict)
    return &dict, err
}
```

### Using Dictionary for Translation
```go
func TranslateWord(dict *Dictionary, word, targetLang string) (string, bool) {
    if translations, exists := dict.Words[word]; exists {
        if translation, found := translations[targetLang]; found {
            return translation, true
        }
    }
    return "", false
}
```

## 📈 Benefits

### Performance
- **Fast Lookups**: O(1) dictionary access
- **No API Calls**: Offline translation capability
- **Reduced Latency**: Instant responses for common words

### Reliability
- **Offline Support**: Works without internet connection
- **Consistent Results**: Same translation every time
- **No Rate Limits**: Unlimited lookups

### Cost Efficiency
- **No API Costs**: Free translations for common words
- **Reduced API Usage**: Only use API for complex translations
- **Bandwidth Savings**: No network requests for common words

## 🔄 Maintenance

### Adding New Words
1. Open the appropriate JSON file
2. Add new word with translations in all languages
3. Update version number in metadata
4. Test the new translations

### Updating Existing Words
1. Locate the word in the dictionary
2. Update translations as needed
3. Update version number
4. Verify all language translations are correct

### Quality Assurance
- Verify translations with native speakers
- Check for consistency across dictionaries
- Ensure proper Unicode encoding
- Test with different input formats

## 🌐 Language Support

### Current Languages
- **English (en)**: Base language
- **Hindi (hi)**: हिन्दी - Devanagari script
- **Marathi (mr)**: मराठी - Devanagari script  
- **Gujarati (gu)**: ગુજરાતી - Gujarati script

### Script Information
- **Devanagari**: Used for Hindi and Marathi
- **Gujarati**: Used for Gujarati
- **Latin**: Used for English

## 📝 Best Practices

### Dictionary Management
- Keep translations consistent across all files
- Use proper Unicode encoding
- Maintain version control
- Document changes in metadata

### Performance Optimization
- Load dictionaries at startup
- Use efficient data structures
- Cache frequently accessed words
- Implement fallback mechanisms

### Quality Control
- Regular review by native speakers
- Automated testing for consistency
- Validation of Unicode characters
- Cross-reference with official sources

---

**Note**: These dictionaries are maintained manually and should be updated regularly to ensure accuracy and completeness.
