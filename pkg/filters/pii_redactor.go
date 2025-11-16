package filters

import (
	"regexp"
	"strings"
	"sync"
)

// PIIRedactor provides detection and redaction of personally identifiable information.
type PIIRedactor struct {
	patterns map[string]*regexp.Regexp
	mu       sync.RWMutex
}

// NewPIIRedactor creates a PIIRedactor with common PII patterns.
func NewPIIRedactor() *PIIRedactor {
	pr := &PIIRedactor{
		patterns: make(map[string]*regexp.Regexp),
	}

	// Common PII patterns
	pr.AddPattern("ssn", regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`))
	pr.AddPattern("email", regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`))
	pr.AddPattern("credit_card", regexp.MustCompile(`\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`))
	pr.AddPattern("phone", regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`))
	pr.AddPattern("ipv4", regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`))
	pr.AddPattern("ipv6", regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`))

	return pr
}

// AddPattern adds a custom PII detection pattern.
func (pr *PIIRedactor) AddPattern(name string, pattern *regexp.Regexp) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.patterns[name] = pattern
}

// RemovePattern removes a PII detection pattern.
func (pr *PIIRedactor) RemovePattern(name string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	delete(pr.patterns, name)
}

// Redact replaces all detected PII with the given replacement string.
func (pr *PIIRedactor) Redact(input string, replacement string) string {
	if replacement == "" {
		replacement = "[REDACTED]"
	}

	pr.mu.RLock()
	defer pr.mu.RUnlock()

	result := input
	for _, pattern := range pr.patterns {
		result = pattern.ReplaceAllString(result, replacement)
	}

	return result
}

// RedactWithTags replaces PII with tagged replacements indicating the type.
func (pr *PIIRedactor) RedactWithTags(input string) string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	result := input
	for name, pattern := range pr.patterns {
		replacement := "[" + strings.ToUpper(name) + "]"
		result = pattern.ReplaceAllString(result, replacement)
	}

	return result
}

// DetectPII returns a map of PII types detected in the input.
func (pr *PIIRedactor) DetectPII(input string) map[string]int {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	detected := make(map[string]int)

	for name, pattern := range pr.patterns {
		matches := pattern.FindAllString(input, -1)
		if len(matches) > 0 {
			detected[name] = len(matches)
		}
	}

	return detected
}

// RedactMap applies redaction to all string values in a map (useful for JSON logs).
func (pr *PIIRedactor) RedactMap(data map[string]interface{}, replacement string) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = pr.Redact(v, replacement)
		case map[string]interface{}:
			result[key] = pr.RedactMap(v, replacement)
		case []interface{}:
			result[key] = pr.redactSlice(v, replacement)
		default:
			result[key] = value
		}
	}

	return result
}

func (pr *PIIRedactor) redactSlice(data []interface{}, replacement string) []interface{} {
	result := make([]interface{}, len(data))

	for i, value := range data {
		switch v := value.(type) {
		case string:
			result[i] = pr.Redact(v, replacement)
		case map[string]interface{}:
			result[i] = pr.RedactMap(v, replacement)
		case []interface{}:
			result[i] = pr.redactSlice(v, replacement)
		default:
			result[i] = value
		}
	}

	return result
}

// PrebuiltRedactors provides common redactor configurations.
var (
	// FullRedactor includes all PII patterns.
	FullRedactor = NewPIIRedactor()

	// NetworkRedactor includes only network-related patterns (IPs).
	NetworkRedactor = func() *PIIRedactor {
		pr := &PIIRedactor{patterns: make(map[string]*regexp.Regexp)}
		pr.AddPattern("ipv4", regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`))
		pr.AddPattern("ipv6", regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`))
		return pr
	}()

	// FinancialRedactor includes financial PII patterns.
	FinancialRedactor = func() *PIIRedactor {
		pr := &PIIRedactor{patterns: make(map[string]*regexp.Regexp)}
		pr.AddPattern("ssn", regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`))
		pr.AddPattern("credit_card", regexp.MustCompile(`\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`))
		return pr
	}()
)
