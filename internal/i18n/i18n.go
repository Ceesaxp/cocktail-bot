package i18n

import (
	"strings"
	"sync"

	"github.com/ceesaxp/cocktail-bot/internal/config"
)

// DefaultLanguage is the language to use when the user's preferred language is not available
const DefaultLanguage = "en"

// Translator handles message translations
type Translator struct {
	translations map[string]map[string]string // language -> key -> text
	fallback     string                       // fallback language
	mutex        sync.RWMutex                 // to ensure thread safety
	config       *config.Config               // application configuration
}

// New creates a new Translator with the specified fallback language
func New(fallbackLang string) *Translator {
	return &Translator{
		translations: make(map[string]map[string]string),
		fallback:     fallbackLang,
	}
}

// NewWithConfig creates a new Translator using configuration
func NewWithConfig(cfg *config.Config) *Translator {
	return &Translator{
		translations: make(map[string]map[string]string),
		fallback:     cfg.GetDefaultLanguage(),
		config:       cfg,
	}
}

// LoadTranslations initializes translations for a language
func (t *Translator) LoadTranslations(lang string, messages map[string]string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// If config is provided, check if language is enabled
	if t.config != nil && !t.config.IsLanguageEnabled(lang) {
		return
	}

	if _, exists := t.translations[lang]; !exists {
		t.translations[lang] = make(map[string]string)
	}

	for key, message := range messages {
		t.translations[lang][key] = message
	}
}

// T returns the translated text for the specified key in the specified language
// If the translation is not available, it returns the text in the fallback language
// If the fallback translation is not available, it returns the key itself
func (t *Translator) T(lang string, key string, args ...string) string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	lang = strings.ToLower(lang)
	
	// Check if the language is available
	translations, langExists := t.translations[lang]
	if !langExists {
		// If the language doesn't exist, use the fallback language
		translations, langExists = t.translations[t.fallback]
		if !langExists {
			return key
		}
	}
	
	// Check if the key exists for the language
	text, keyExists := translations[key]
	if !keyExists {
		// Try fallback language
		if fallbackTranslations, fbExists := t.translations[t.fallback]; fbExists {
			if fallbackText, fbKeyExists := fallbackTranslations[key]; fbKeyExists {
				text = fallbackText
			} else {
				return key
			}
		} else {
			return key
		}
	}
	
	// Replace arguments in the text
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			placeholder := "{" + args[i] + "}"
			text = strings.ReplaceAll(text, placeholder, args[i+1])
		}
	}
	
	return text
}

// GetAvailableLanguages returns a list of available languages
func (t *Translator) GetAvailableLanguages() []string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	// If config is provided, return enabled languages that are also loaded
	if t.config != nil {
		enabledLangs := t.config.GetEnabledLanguages()
		availableLangs := make([]string, 0, len(enabledLangs))
		
		for _, lang := range enabledLangs {
			if _, exists := t.translations[lang]; exists {
				availableLangs = append(availableLangs, lang)
			}
		}
		
		return availableLangs
	}
	
	// Otherwise, return all loaded languages
	languages := make([]string, 0, len(t.translations))
	for lang := range t.translations {
		languages = append(languages, lang)
	}
	
	return languages
}

// GetFallbackLanguage returns the fallback language
func (t *Translator) GetFallbackLanguage() string {
	return t.fallback
}

// DetectLanguage attempts to detect the language from Telegram's From.LanguageCode
// If it cannot detect or the language is not supported, it returns the default language
func (t *Translator) DetectLanguage(tgLangCode string) string {
	if tgLangCode == "" {
		return t.fallback
	}
	
	langCode := strings.ToLower(tgLangCode)
	
	// Handle special cases where Telegram uses different codes
	// than our translation files might use
	switch {
	case strings.HasPrefix(langCode, "en"):
		langCode = "en"
	case strings.HasPrefix(langCode, "es"):
		langCode = "es"
	case strings.HasPrefix(langCode, "fr"):
		langCode = "fr"
	case strings.HasPrefix(langCode, "de"):
		langCode = "de"
	case strings.HasPrefix(langCode, "it"):
		langCode = "it"
	case strings.HasPrefix(langCode, "ru"):
		langCode = "ru"
	case strings.HasPrefix(langCode, "sr"):
		langCode = "sr"
	case strings.HasPrefix(langCode, "zh"):
		langCode = "zh"
	}
	
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	
	// If config is provided, check if language is enabled
	if t.config != nil && !t.config.IsLanguageEnabled(langCode) {
		return t.fallback
	}
	
	if _, exists := t.translations[langCode]; exists {
		return langCode
	}
	
	return t.fallback
}