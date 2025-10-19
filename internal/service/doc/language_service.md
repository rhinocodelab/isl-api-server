# Language Service - Detailed Workflow Documentation

## 📋 Overview

The `LanguageService` is a utility service that handles language validation and management for the ISL API Server. This service provides language code validation, target language resolution, and language management capabilities for the text translation system.

## 🏗️ Service Architecture

### Service Structure

```go
type LanguageService struct {
    supportedLanguages []string  // List of supported language codes
}
```

**Components:**
- **`supportedLanguages`**: Array of supported language codes (en, hi, mr, gu)
- **No external dependencies**: Pure utility service
- **Stateless operations**: All methods are stateless and thread-safe

---

## 🔄 Service Initialization Flow

### Constructor: `NewLanguageService()`

```go
func NewLanguageService() *LanguageService {
    return &LanguageService{
        supportedLanguages: []string{"en", "hi", "mr", "gu"},
    }
}
```

### Initialization Process

**Step 1: Service Creation**
- Creates new `LanguageService` instance
- Initializes with predefined supported languages
- No external configuration required

**Step 2: Language Configuration**
```go
supportedLanguages: []string{"en", "hi", "mr", "gu"}
```

**Supported Languages:**
- **`en`**: English
- **`hi`**: Hindi (हिन्दी)
- **`mr`**: Marathi (मराठी)
- **`gu`**: Gujarati (ગુજરાતી)

**Step 3: Service Ready**
- Service is immediately ready for use
- No initialization errors possible
- Thread-safe for concurrent access

---

## 🔍 Language Validation Flow

### Method: `IsValidLanguage()`

```go
func (l *LanguageService) IsValidLanguage(lang string) bool {
    for _, supportedLang := range l.supportedLanguages {
        if lang == supportedLang {
            return true
        }
    }
    return false
}
```

### Step 1: Input Validation
- Receives language code as string parameter
- Case-sensitive comparison
- No preprocessing or normalization

### Step 2: Iterative Comparison
```go
for _, supportedLang := range l.supportedLanguages {
    if lang == supportedLang {
        return true
    }
}
```

**Process:**
- Iterates through all supported languages
- Performs exact string match
- Returns `true` on first match
- Returns `false` if no match found

### Step 3: Result Return
- **Valid languages**: Returns `true`
- **Invalid languages**: Returns `false`
- **Empty string**: Returns `false`
- **Case mismatch**: Returns `false`

### Usage Examples

```go
service := NewLanguageService()

// Valid languages
service.IsValidLanguage("en")  // Returns: true
service.IsValidLanguage("hi") // Returns: true
service.IsValidLanguage("mr") // Returns: true
service.IsValidLanguage("gu") // Returns: true

// Invalid languages
service.IsValidLanguage("fr")  // Returns: false
service.IsValidLanguage("es") // Returns: false
service.IsValidLanguage("")   // Returns: false
service.IsValidLanguage("EN") // Returns: false (case sensitive)
```

---

## 🎯 Target Language Resolution Flow

### Method: `GetTargetLanguages()`

```go
func (l *LanguageService) GetTargetLanguages(sourceLang string) []string {
    var targetLanguages []string
    for _, lang := range l.supportedLanguages {
        if lang != sourceLang {
            targetLanguages = append(targetLanguages, lang)
        }
    }
    return targetLanguages
}
```

### Step 1: Input Processing
- Receives source language as parameter
- No validation of source language (assumes valid input)
- Creates empty slice for target languages

### Step 2: Language Filtering
```go
for _, lang := range l.supportedLanguages {
    if lang != sourceLang {
        targetLanguages = append(targetLanguages, lang)
    }
}
```

**Process:**
- Iterates through all supported languages
- Excludes the source language from results
- Appends remaining languages to target list

### Step 3: Result Return
- Returns slice of target languages
- Always returns 3 languages (4 total - 1 source)
- Order preserved from supported languages list

### Usage Examples

```go
service := NewLanguageService()

// Source: English
targets := service.GetTargetLanguages("en")
// Returns: ["hi", "mr", "gu"]

// Source: Hindi
targets := service.GetTargetLanguages("hi")
// Returns: ["en", "mr", "gu"]

// Source: Marathi
targets := service.GetTargetLanguages("mr")
// Returns: ["en", "hi", "gu"]

// Source: Gujarati
targets := service.GetTargetLanguages("gu")
// Returns: ["en", "hi", "mr"]
```

---

## 📋 Language Information Flow

### Method: `GetSupportedLanguages()`

```go
func (l *LanguageService) GetSupportedLanguages() []string {
    return l.supportedLanguages
}
```

### Step 1: Direct Return
- Returns copy of supported languages slice
- No processing or filtering
- Immediate response

### Step 2: Result Structure
```go
[]string{"en", "hi", "mr", "gu"}
```

**Returns:**
- Complete list of supported languages
- Same order as initialization
- All 4 supported languages

### Usage Examples

```go
service := NewLanguageService()

// Get all supported languages
languages := service.GetSupportedLanguages()
// Returns: ["en", "hi", "mr", "gu"]

// Use for validation
for _, lang := range service.GetSupportedLanguages() {
    fmt.Printf("Supported: %s\n", lang)
}
```

---

## 🔄 Complete API Integration Flow

### 1. Router Request Processing

```go
// In router.go
var req models.TranslationRequest
if err := c.ShouldBindJSON(&req); err != nil {
    // Handle binding error
}

// Validate source language
if !languageService.IsValidLanguage(req.SourceLanguage) {
    c.JSON(http.StatusBadRequest, models.ErrorResponse{
        Error:   "Invalid source language",
        Message: "Source language must be one of: en, hi, mr, gu",
    })
    return
}
```

### 2. Target Language Resolution

```go
// Get target languages
targetLanguages := languageService.GetTargetLanguages(req.SourceLanguage)
// Example: source="en" → targets=["hi", "mr", "gu"]
```

### 3. Translation Service Call

```go
// Translate to all target languages
translations, err := translationService.TranslateToMultipleLanguages(
    ctx, 
    req.Text, 
    req.SourceLanguage, 
    targetLanguages
)
```

### 4. Response Construction

```go
response := models.TranslationResponse{
    OriginalText:   req.Text,
    SourceLanguage: req.SourceLanguage,
    Translations:   translations,
}
```

---

## 🚨 Error Handling Scenarios

### Invalid Source Language

**Request:**
```json
{
  "text": "Hello World",
  "source_language": "fr"
}
```

**Response:**
```json
{
  "error": "Invalid source language",
  "message": "Source language must be one of: en, hi, mr, gu"
}
```

**HTTP Status:** `400 Bad Request`

### Valid Language Flow

**Request:**
```json
{
  "text": "Hello World",
  "source_language": "en"
}
```

**Process:**
1. `IsValidLanguage("en")` → `true`
2. `GetTargetLanguages("en")` → `["hi", "mr", "gu"]`
3. Translation to 3 target languages
4. Success response

---

## 📊 Language Mapping Reference

### Language Codes

| Code | Language | Native Name | Script |
|------|----------|-------------|---------|
| `en` | English | English | Latin |
| `hi` | Hindi | हिन्दी | Devanagari |
| `mr` | Marathi | मराठी | Devanagari |
| `gu` | Gujarati | ગુજરાતી | Gujarati |

### Translation Matrix

| Source | Targets |
|--------|---------|
| `en` | `hi`, `mr`, `gu` |
| `hi` | `en`, `mr`, `gu` |
| `mr` | `en`, `hi`, `gu` |
| `gu` | `en`, `hi`, `mr` |

---

## 🔧 Configuration and Customization

### Adding New Languages

To add new languages, modify the `NewLanguageService()` function:

```go
func NewLanguageService() *LanguageService {
    return &LanguageService{
        supportedLanguages: []string{"en", "hi", "mr", "gu", "bn", "ta"},
    }
}
```

### Language Validation Rules

- **Case Sensitive**: "en" ≠ "EN"
- **Exact Match**: No partial matching
- **No Normalization**: Input used as-is
- **No Whitespace Handling**: Spaces not trimmed

---

## 🚀 Performance Characteristics

### Time Complexity
- **`IsValidLanguage()`**: O(n) where n = number of supported languages
- **`GetTargetLanguages()`**: O(n) where n = number of supported languages
- **`GetSupportedLanguages()`**: O(1) - direct return

### Space Complexity
- **Service Instance**: O(n) where n = number of supported languages
- **Method Calls**: O(1) additional space
- **Memory Efficient**: No dynamic allocations in methods

### Optimization Notes
- **Small Language Set**: Only 4 languages, linear search is efficient
- **No Caching Needed**: Simple operations, no expensive computations
- **Thread Safe**: No shared state modifications

---

## 🧪 Testing Scenarios

### Unit Test Examples

```go
func TestLanguageService(t *testing.T) {
    service := NewLanguageService()
    
    // Test valid languages
    assert.True(t, service.IsValidLanguage("en"))
    assert.True(t, service.IsValidLanguage("hi"))
    assert.True(t, service.IsValidLanguage("mr"))
    assert.True(t, service.IsValidLanguage("gu"))
    
    // Test invalid languages
    assert.False(t, service.IsValidLanguage("fr"))
    assert.False(t, service.IsValidLanguage(""))
    assert.False(t, service.IsValidLanguage("EN"))
    
    // Test target language resolution
    targets := service.GetTargetLanguages("en")
    assert.Equal(t, []string{"hi", "mr", "gu"}, targets)
    
    // Test supported languages
    supported := service.GetSupportedLanguages()
    assert.Equal(t, []string{"en", "hi", "mr", "gu"}, supported)
}
```

### Edge Cases

```go
// Empty string
service.IsValidLanguage("") // Returns: false

// Case sensitivity
service.IsValidLanguage("EN") // Returns: false

// Special characters
service.IsValidLanguage("en-") // Returns: false

// Unicode
service.IsValidLanguage("हिन्दी") // Returns: false (not language code)
```

---

## 📝 Integration with Translation Service

### Workflow Integration

1. **Request Validation**
   ```go
   if !languageService.IsValidLanguage(req.SourceLanguage) {
       // Return error response
   }
   ```

2. **Target Resolution**
   ```go
   targetLanguages := languageService.GetTargetLanguages(req.SourceLanguage)
   ```

3. **Translation Execution**
   ```go
   translations, err := translationService.TranslateToMultipleLanguages(
       ctx, req.Text, req.SourceLanguage, targetLanguages
   )
   ```

### Service Dependencies

- **No External Dependencies**: Pure utility service
- **Used By**: Router for request validation
- **Uses**: No other services
- **Thread Safe**: Can be used concurrently

---

## 🔍 Monitoring and Debugging

### Key Metrics to Monitor
- Language validation success rate
- Most requested source languages
- Target language distribution
- Invalid language request frequency

### Debug Information
- Language validation results
- Target language resolution
- Supported language queries
- Error patterns in language requests

### Logging Integration
```go
// In router.go
if !languageService.IsValidLanguage(req.SourceLanguage) {
    logger.Error("Invalid language requested", "language", req.SourceLanguage)
    // Return error response
}
```

---

## 🎯 Best Practices

### Service Usage
- **Single Instance**: Create once, reuse everywhere
- **Stateless Operations**: No need to reset or reinitialize
- **Error Handling**: Always validate before translation
- **Performance**: Service is lightweight, no optimization needed

### Language Code Standards
- **Use ISO 639-1 codes**: Standard 2-letter codes
- **Case Sensitivity**: Always use lowercase
- **Validation**: Validate before processing
- **Documentation**: Keep language list updated

This service provides a robust, efficient, and maintainable language management solution for the ISL API Server! 🚀
