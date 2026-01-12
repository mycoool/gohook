package i18n

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// MessageID is the unique identifier for a translatable message
type MessageID string

// Locale represents a language locale (e.g., "en", "zh", "en-US")
type Locale string

const (
	// LocaleEnglish is the English locale
	LocaleEnglish Locale = "en"
	// LocaleChinese is the Chinese locale
	LocaleChinese Locale = "zh"
)

// Translator handles message translation
type Translator struct {
	mu             sync.RWMutex
	locale         Locale
	messages       map[Locale]map[MessageID]string
	fallbackLocale Locale
}

var (
	globalTranslator *Translator
	once             sync.Once
)

// TranslationFile represents the structure of locale JSON files
type TranslationFile struct {
	Messages map[string]string `json:"messages"`
}

// Init initializes the global translator with the specified locale
func Init(locale Locale) {
	once.Do(func() {
		globalTranslator = NewTranslator(locale)
	})
}

// GetGlobal returns the global translator instance
func GetGlobal() *Translator {
	if globalTranslator == nil {
		globalTranslator = NewTranslator(LocaleEnglish)
	}
	return globalTranslator
}

// NewTranslator creates a new translator instance
func NewTranslator(locale Locale) *Translator {
	t := &Translator{
		locale:         locale,
		messages:       make(map[Locale]map[MessageID]string),
		fallbackLocale: LocaleEnglish,
	}

	// Load translation files
	if err := t.loadTranslations(); err != nil {
		log.Printf("Warning: Failed to load translations: %v", err)
	}

	return t
}

// loadTranslations loads all translation files from the locales directory
func (t *Translator) loadTranslations() error {
	localesDir := filepath.Join("internal", "i18n", "locales")

	// Check if directory exists
	if _, err := os.Stat(localesDir); os.IsNotExist(err) {
		return fmt.Errorf("locales directory not found: %s", localesDir)
	}

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return fmt.Errorf("failed to read locales directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		ext := filepath.Ext(filename)
		if ext != ".json" {
			continue
		}

		// Extract locale from filename (e.g., "en.json" -> "en")
		localeStr := filename[:len(filename)-len(ext)]
		locale := Locale(localeStr)

		filePath := filepath.Join(localesDir, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Warning: Failed to read translation file %s: %v", filePath, err)
			continue
		}

		var tf TranslationFile
		if err := json.Unmarshal(data, &tf); err != nil {
			log.Printf("Warning: Failed to parse translation file %s: %v", filePath, err)
			continue
		}

		// Convert string keys to MessageID
		messages := make(map[MessageID]string)
		for key, value := range tf.Messages {
			messages[MessageID(key)] = value
		}

		t.mu.Lock()
		t.messages[locale] = messages
		t.mu.Unlock()

		log.Printf("Loaded %d translations for locale %s from %s", len(messages), locale, filePath)
	}

	return nil
}

// SetLocale changes the current locale
func (t *Translator) SetLocale(locale Locale) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.locale = locale
}

// GetLocale returns the current locale
func (t *Translator) GetLocale() Locale {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.locale
}

// T translates a message ID to the current locale
// Parameters are substituted using fmt.Sprintf
func (t *Translator) T(msgID MessageID, args ...interface{}) string {
	return t.TranslateWithLocale(t.GetLocale(), msgID, args...)
}

// TranslateWithLocale translates a message ID to a specific locale
func (t *Translator) TranslateWithLocale(locale Locale, msgID MessageID, args ...interface{}) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Try to get message for the requested locale
	if messages, ok := t.messages[locale]; ok {
		if msg, ok := messages[msgID]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(msg, args...)
			}
			return msg
		}
	}

	// Fallback to English locale
	if locale != t.fallbackLocale {
		if messages, ok := t.messages[t.fallbackLocale]; ok {
			if msg, ok := messages[msgID]; ok {
				if len(args) > 0 {
					return fmt.Sprintf(msg, args...)
				}
				return msg
			}
		}
	}

	// Return message ID as last resort
	if len(args) > 0 {
		return fmt.Sprintf(string(msgID), args...)
	}
	return string(msgID)
}

// Helper functions for common logging scenarios

// Info returns an informational message in the current locale
func (t *Translator) Info(msgID MessageID, args ...interface{}) string {
	return t.T(msgID, args...)
}

// Error returns an error message in the current locale
func (t *Translator) Error(msgID MessageID, args ...interface{}) string {
	return t.T(msgID, args...)
}

// Warning returns a warning message in the current locale
func (t *Translator) Warning(msgID MessageID, args ...interface{}) string {
	return t.T(msgID, args...)
}

// Debug returns a debug message in the current locale
func (t *Translator) Debug(msgID MessageID, args ...interface{}) string {
	return t.T(msgID, args...)
}

// Global helper functions (use the global translator)

// T translates a message using the global translator
func T(msgID MessageID, args ...interface{}) string {
	return GetGlobal().T(msgID, args...)
}

// SetLocale sets the locale for the global translator
func SetLocale(locale Locale) {
	GetGlobal().SetLocale(locale)
}

// GetLocale returns the current locale from the global translator
func GetLocale() Locale {
	return GetGlobal().GetLocale()
}
