# Text Translation Models - Detailed Workflow Documentation

## 📋 Overview

The `text_translation.go` file defines the data models and structures used for the ISL API Server's text translation functionality. These models handle request/response serialization, validation, and data transfer between the API endpoints and the translation services.

## 🏗️ Model Architecture

### File Structure
```go
package models

// TranslationRequest - Input model
// TranslationResponse - Output model  
// ErrorResponse - Error model
```

**Components:**
- **Request Models**: Handle incoming API requests
- **Response Models**: Structure API responses
- **Error Models**: Standardize error responses
- **JSON Serialization**: Automatic JSON marshaling/unmarshaling

---

## 📥 TranslationRequest Model

### Structure Definition

```go
type TranslationRequest struct {
    Text           string `json:"text" binding:"required"`
    SourceLanguage string `json:"source_language" binding:"required"`
}
```

### Field Analysis

#### **`Text` Field**
```go
Text string `json:"text" binding:"required"`
```

**Purpose:**
- Contains the text to be translated
- User input for translation processing
- Required field for API validation

**Characteristics:**
- **Type**: `string` - Unicode text support
- **JSON Tag**: `"text"` - API field name
- **Validation**: `binding:"required"` - Gin validation
- **Constraints**: No length limits defined
- **Encoding**: UTF-8 support for all languages

**Usage Examples:**
```json
{
  "text": "Hello World",
  "source_language": "en"
}
```

```json
{
  "text": "नमस्ते दुनिया",
  "source_language": "hi"
}
```

#### **`SourceLanguage` Field**
```go
SourceLanguage string `json:"source_language" binding:"required"`
```

**Purpose:**
- Specifies the source language of the input text
- Used for translation direction determination
- Required for API validation

**Characteristics:**
- **Type**: `string` - Language code
- **JSON Tag**: `"source_language"` - API field name
- **Validation**: `binding:"required"` - Gin validation
- **Format**: ISO 639-1 language codes
- **Supported Values**: "en", "hi", "mr", "gu"

**Usage Examples:**
```json
{
  "text": "Hello World",
  "source_language": "en"
}
```

```json
{
  "text": "नमस्ते दुनिया", 
  "source_language": "hi"
}
```

### Request Validation Flow

#### **1. JSON Binding**
```go
var req models.TranslationRequest
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, models.ErrorResponse{
        Error:   "Invalid request",
        Message: err.Error(),
    })
    return
}
```

**Process:**
- Gin framework automatically binds JSON to struct
- Validates required fields
- Returns binding errors if validation fails
- Handles malformed JSON gracefully

#### **2. Field Validation**
- **Required Fields**: Both `text` and `source_language` must be present
- **Type Validation**: Fields must be strings
- **Empty String Handling**: Empty strings are considered valid (business logic validation needed)

#### **3. Business Logic Validation**
```go
// Additional validation in router
if !languageService.IsValidLanguage(req.SourceLanguage) {
    c.JSON(http.StatusBadRequest, models.ErrorResponse{
        Error:   "Invalid source language",
        Message: "Source language must be one of: en, hi, mr, gu",
    })
    return
}
```

### Request Examples

#### **Valid Requests**
```json
{
  "text": "Hello World",
  "source_language": "en"
}
```

```json
{
  "text": "नमस्ते दुनिया",
  "source_language": "hi"
}
```

```json
{
  "text": "हॅलो वर्ल्ड",
  "source_language": "mr"
}
```

#### **Invalid Requests**
```json
{
  "text": "Hello World"
  // Missing source_language
}
```

```json
{
  "source_language": "en"
  // Missing text
}
```

```json
{
  "text": "Hello World",
  "source_language": "fr"  // Unsupported language
}
```

---

## 📤 TranslationResponse Model

### Structure Definition

```go
type TranslationResponse struct {
    OriginalText   string            `json:"original_text"`
    SourceLanguage string            `json:"source_language"`
    Translations   map[string]string `json:"translations"`
}
```

### Field Analysis

#### **`OriginalText` Field**
```go
OriginalText string `json:"original_text"`
```

**Purpose:**
- Echoes back the original input text
- Provides context for the translations
- Helps clients match responses to requests

**Characteristics:**
- **Type**: `string` - Unicode text
- **JSON Tag**: `"original_text"` - API field name
- **Source**: Copied from request `Text` field
- **Encoding**: UTF-8 preserved from input

**Example:**
```json
{
  "original_text": "Hello World",
  "source_language": "en",
  "translations": {
    "hi": "नमस्ते दुनिया",
    "mr": "हॅलो वर्ल्ड",
    "gu": "હેલો વર્લ્ડ"
  }
}
```

#### **`SourceLanguage` Field**
```go
SourceLanguage string `json:"source_language"`
```

**Purpose:**
- Echoes back the source language code
- Provides context for translation direction
- Helps clients understand the translation flow

**Characteristics:**
- **Type**: `string` - Language code
- **JSON Tag**: `"source_language"` - API field name
- **Source**: Copied from request `SourceLanguage` field
- **Values**: "en", "hi", "mr", "gu"

**Example:**
```json
{
  "original_text": "Hello World",
  "source_language": "en",
  "translations": { ... }
}
```

#### **`Translations` Field**
```go
Translations map[string]string `json:"translations"`
```

**Purpose:**
- Contains all translation results
- Key-value pairs of language codes and translated text
- Excludes the source language from results

**Characteristics:**
- **Type**: `map[string]string` - Key-value mapping
- **JSON Tag**: `"translations"` - API field name
- **Keys**: Target language codes ("hi", "mr", "gu" for source "en")
- **Values**: Translated text in target languages
- **Size**: Always 3 entries (4 total languages - 1 source)

**Example:**
```json
{
  "original_text": "Hello World",
  "source_language": "en",
  "translations": {
    "hi": "नमस्ते दुनिया",
    "mr": "हॅलो वर्ल्ड", 
    "gu": "હેલો વર્લ્ડ"
  }
}
```

### Response Construction Flow

#### **1. Service Call**
```go
// Get target languages
targetLanguages := languageService.GetTargetLanguages(req.SourceLanguage)
// Example: source="en" → targets=["hi", "mr", "gu"]

// Translate to all target languages
translations, err := translationService.TranslateToMultipleLanguages(
    ctx, req.Text, req.SourceLanguage, targetLanguages
)
```

#### **2. Response Assembly**
```go
response := models.TranslationResponse{
    OriginalText:   req.Text,
    SourceLanguage: req.SourceLanguage,
    Translations:   translations,
}
```

#### **3. JSON Serialization**
```go
c.JSON(http.StatusOK, response)
```

**Process:**
- Gin automatically serializes struct to JSON
- Preserves Unicode characters
- Maintains field order
- Handles special characters correctly

### Response Examples

#### **English to Other Languages**
```json
{
  "original_text": "Hello World",
  "source_language": "en",
  "translations": {
    "hi": "नमस्ते दुनिया",
    "mr": "हॅलो वर्ल्ड",
    "gu": "હેલો વર્લ્ડ"
  }
}
```

#### **Hindi to Other Languages**
```json
{
  "original_text": "नमस्ते दुनिया",
  "source_language": "hi", 
  "translations": {
    "en": "Hello World",
    "mr": "हॅलो वर्ल्ड",
    "gu": "હેલો વર્લ્ડ"
  }
}
```

#### **Marathi to Other Languages**
```json
{
  "original_text": "हॅलो वर्ल्ड",
  "source_language": "mr",
  "translations": {
    "en": "Hello World",
    "hi": "नमस्ते दुनिया", 
    "gu": "હેલો વર્લ્ડ"
  }
}
```

---

## ❌ ErrorResponse Model

### Structure Definition

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message"`
}
```

### Field Analysis

#### **`Error` Field**
```go
Error string `json:"error"`
```

**Purpose:**
- Provides a short, machine-readable error identifier
- Used for error categorization
- Helps clients handle different error types

**Characteristics:**
- **Type**: `string` - Error identifier
- **JSON Tag**: `"error"` - API field name
- **Format**: Short, descriptive error codes
- **Examples**: "Invalid request", "Translation failed", "Service unavailable"

#### **`Message` Field**
```go
Message string `json:"message"`
```

**Purpose:**
- Provides detailed, human-readable error description
- Explains what went wrong and how to fix it
- Helps developers and users understand errors

**Characteristics:**
- **Type**: `string` - Error description
- **JSON Tag**: `"message"` - API field name
- **Format**: Detailed, actionable error messages
- **Length**: Can be longer than error field

### Error Response Examples

#### **Validation Errors**
```json
{
  "error": "Invalid request",
  "message": "Key: 'TranslationRequest.Text' Error:Field validation for 'Text' failed on the 'required' tag"
}
```

```json
{
  "error": "Invalid source language",
  "message": "Source language must be one of: en, hi, mr, gu"
}
```

#### **Service Errors**
```json
{
  "error": "Translation service unavailable",
  "message": "GCP Translation service is not configured. Please set GCP_PROJECT_ID environment variable."
}
```

```json
{
  "error": "Translation failed",
  "message": "Unable to translate text. Please try again."
}
```

#### **System Errors**
```json
{
  "error": "Internal Server Error",
  "message": "An unexpected error occurred while processing your request"
}
```

### Error Handling Flow

#### **1. Request Validation Errors**
```go
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, models.ErrorResponse{
        Error:   "Invalid request",
        Message: err.Error(),
    })
    return
}
```

#### **2. Business Logic Errors**
```go
if !languageService.IsValidLanguage(req.SourceLanguage) {
    c.JSON(http.StatusBadRequest, models.ErrorResponse{
        Error:   "Invalid source language",
        Message: "Source language must be one of: en, hi, mr, gu",
    })
    return
}
```

#### **3. Service Errors**
```go
if translationService == nil {
    c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
        Error:   "Translation service unavailable",
        Message: "GCP Translation service is not configured. Please set GCP_PROJECT_ID environment variable.",
    })
    return
}
```

#### **4. Translation Errors**
```go
if err != nil {
    logger.Error("Translation failed", "error", err)
    c.JSON(http.StatusInternalServerError, models.ErrorResponse{
        Error:   "Translation failed",
        Message: "Unable to translate text. Please try again.",
    })
    return
}
```

---

## 🔄 Complete API Workflow

### 1. Request Processing

```bash
POST /api/v1/text-translate
Content-Type: application/json

{
  "text": "Hello World",
  "source_language": "en"
}
```

### 2. Model Binding

```go
var req models.TranslationRequest
// Gin automatically binds JSON to struct
// Validates required fields
// Returns binding errors if validation fails
```

### 3. Business Logic Processing

```go
// Validate source language
if !languageService.IsValidLanguage(req.SourceLanguage) {
    // Return error response
}

// Get target languages
targetLanguages := languageService.GetTargetLanguages(req.SourceLanguage)

// Perform translation
translations, err := translationService.TranslateToMultipleLanguages(
    ctx, req.Text, req.SourceLanguage, targetLanguages
)
```

### 4. Response Construction

```go
response := models.TranslationResponse{
    OriginalText:   req.Text,
    SourceLanguage: req.SourceLanguage,
    Translations:   translations,
}

c.JSON(http.StatusOK, response)
```

### 5. JSON Response

```json
{
  "original_text": "Hello World",
  "source_language": "en",
  "translations": {
    "hi": "नमस्ते दुनिया",
    "mr": "हॅलो वर्ल्ड",
    "gu": "હેલો વર્લ્ડ"
  }
}
```

---

## 🧪 Testing Scenarios

### Request Validation Tests

```go
func TestTranslationRequest(t *testing.T) {
    // Valid request
    req := models.TranslationRequest{
        Text:           "Hello World",
        SourceLanguage: "en",
    }
    assert.Equal(t, "Hello World", req.Text)
    assert.Equal(t, "en", req.SourceLanguage)
    
    // Empty text (valid struct, invalid business logic)
    req = models.TranslationRequest{
        Text:           "",
        SourceLanguage: "en",
    }
    assert.Equal(t, "", req.Text)
}
```

### Response Construction Tests

```go
func TestTranslationResponse(t *testing.T) {
    translations := map[string]string{
        "hi": "नमस्ते दुनिया",
        "mr": "हॅलो वर्ल्ड",
        "gu": "હેલો વર્લ્ડ",
    }
    
    response := models.TranslationResponse{
        OriginalText:   "Hello World",
        SourceLanguage: "en",
        Translations:   translations,
    }
    
    assert.Equal(t, "Hello World", response.OriginalText)
    assert.Equal(t, "en", response.SourceLanguage)
    assert.Equal(t, 3, len(response.Translations))
    assert.Equal(t, "नमस्ते दुनिया", response.Translations["hi"])
}
```

### Error Response Tests

```go
func TestErrorResponse(t *testing.T) {
    err := models.ErrorResponse{
        Error:   "Invalid request",
        Message: "Missing required field: text",
    }
    
    assert.Equal(t, "Invalid request", err.Error)
    assert.Equal(t, "Missing required field: text", err.Message)
}
```

---

## 📊 JSON Serialization Details

### Request Serialization

```go
// Go struct to JSON
req := models.TranslationRequest{
    Text:           "Hello World",
    SourceLanguage: "en",
}

// JSON output
{
  "text": "Hello World",
  "source_language": "en"
}
```

### Response Serialization

```go
// Go struct to JSON
response := models.TranslationResponse{
    OriginalText:   "Hello World",
    SourceLanguage: "en",
    Translations: map[string]string{
        "hi": "नमस्ते दुनिया",
        "mr": "हॅलो वर्ल्ड",
        "gu": "હેલો વર્લ્ડ",
    },
}

// JSON output
{
  "original_text": "Hello World",
  "source_language": "en",
  "translations": {
    "hi": "नमस्ते दुनिया",
    "mr": "हॅलो वर्ल्ड",
    "gu": "હેલો વર્લ્ડ"
  }
}
```

### Unicode Handling

- **UTF-8 Support**: All models support Unicode characters
- **Language Scripts**: Devanagari (Hindi, Marathi), Gujarati, Latin (English)
- **JSON Encoding**: Automatic UTF-8 encoding in JSON responses
- **Character Preservation**: No character loss during serialization

---

## 🔧 Model Customization

### Adding New Fields

```go
type TranslationRequest struct {
    Text           string `json:"text" binding:"required"`
    SourceLanguage string `json:"source_language" binding:"required"`
    Context        string `json:"context,omitempty"`  // New optional field
}
```

### Adding Validation Tags

```go
type TranslationRequest struct {
    Text           string `json:"text" binding:"required,min=1,max=1000"`
    SourceLanguage string `json:"source_language" binding:"required,oneof=en hi mr gu"`
}
```

### Custom JSON Tags

```go
type TranslationResponse struct {
    OriginalText   string            `json:"original_text"`
    SourceLanguage string            `json:"source_language"`
    Translations   map[string]string `json:"translations"`
    Timestamp      time.Time         `json:"timestamp,omitempty"`
}
```

---

## 🚀 Performance Considerations

### Memory Usage
- **Request Models**: Minimal memory footprint
- **Response Models**: Scales with translation count
- **Error Models**: Very lightweight
- **JSON Marshaling**: Efficient Go JSON library

### Serialization Performance
- **Request Binding**: Fast JSON unmarshaling
- **Response Serialization**: Fast JSON marshaling
- **Unicode Handling**: Efficient UTF-8 processing
- **Map Operations**: O(1) access for translations

### Optimization Tips
- **Reuse Models**: Create once, reuse for multiple requests
- **Avoid Deep Copying**: Use pointers for large responses
- **JSON Streaming**: For very large responses, consider streaming
- **Validation Caching**: Cache validation results for repeated requests

This comprehensive model system provides robust, efficient, and maintainable data structures for the ISL API Server's text translation functionality! 🚀
