package filters

import (
	"regexp"
	"strings"
)

// UniversalKVParser parses any syslog message containing key=value pairs.
// It's a general-purpose parser that works with most firewall, network device,
// and security appliance log formats.
type UniversalKVParser struct {
	// kvRegex matches key=value patterns with optional quoted values
	kvRegex *regexp.Regexp
	// separators are the characters that separate key=value pairs
	separators string
}

// NewUniversalKVParser creates a new parser instance
func NewUniversalKVParser() *UniversalKVParser {
	// Match patterns like:
	// key=value
	// key="quoted value"
	// key='quoted value'
	// Also handles keys with dots, dashes, underscores
	kvRegex := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_\-.]*)\s*=\s*(?:"([^"]*?)"|'([^']*?)'|([^\s,;|]+))`)

	return &UniversalKVParser{
		kvRegex:    kvRegex,
		separators: " \t,;|",
	}
}

// Parse extracts key-value pairs from a log message and adds them to the event map.
// It handles:
// - Unquoted values: key=value
// - Double-quoted values: key="some value"
// - Single-quoted values: key='some value'
// - Various delimiters: spaces, commas, semicolons, pipes
func (p *UniversalKVParser) Parse(event map[string]interface{}) error {
	// Get the raw message from common field names
	var raw string
	for _, field := range []string{"message", "raw", "msg", "log", "syslog_message", "content"} {
		if v, ok := event[field]; ok {
			if s, ok := v.(string); ok {
				raw = s
				break
			}
		}
	}

	if raw == "" {
		return nil // Nothing to parse
	}

	// Extract all key=value pairs
	matches := p.kvRegex.FindAllStringSubmatch(raw, -1)

	extracted := make(map[string]string)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		key := strings.TrimSpace(match[1])
		if key == "" {
			continue
		}

		// Value is in group 2 (double-quoted), 3 (single-quoted), or 4 (unquoted)
		var value string
		if match[2] != "" {
			value = match[2] // Double-quoted
		} else if match[3] != "" {
			value = match[3] // Single-quoted
		} else if match[4] != "" {
			value = match[4] // Unquoted
		}

		// Normalize the key (lowercase with underscores)
		normalizedKey := p.normalizeKey(key)
		extracted[normalizedKey] = value
	}

	// Add extracted fields to event
	for k, v := range extracted {
		// Don't overwrite existing fields
		if _, exists := event[k]; !exists {
			event[k] = v
		}
	}

	// Add a flag indicating this was KV-parsed
	if len(extracted) > 0 {
		event["_kv_parsed"] = true
		event["_kv_field_count"] = len(extracted)
	}

	// Try to detect and normalize common fields for severity routing
	p.normalizeSeverity(event, extracted)

	return nil
}

// normalizeKey converts a key to lowercase with underscores
func (p *UniversalKVParser) normalizeKey(key string) string {
	// Replace common separators with underscores
	result := strings.ToLower(key)
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, ".", "_")
	return result
}

// normalizeSeverity attempts to detect severity from common field patterns
// and normalizes it to a standard format for routing
func (p *UniversalKVParser) normalizeSeverity(event map[string]interface{}, extracted map[string]string) {
	// Common severity field names
	severityFields := []string{
		"severity", "sev", "level", "priority", "pri", "risk",
		"threat_level", "alert_level", "log_level", "event_severity",
	}

	var severityValue string
	for _, field := range severityFields {
		if v, ok := extracted[field]; ok {
			severityValue = strings.ToLower(v)
			break
		}
	}

	if severityValue == "" {
		return
	}

	// Map to standard severity levels
	var normalized string
	switch {
	case containsAny(severityValue, "crit", "fatal", "emergency", "emerg", "0", "1"):
		normalized = "critical"
	case containsAny(severityValue, "high", "error", "err", "alert", "2", "3"):
		normalized = "high"
	case containsAny(severityValue, "med", "warn", "warning", "4", "5"):
		normalized = "medium"
	case containsAny(severityValue, "low", "info", "notice", "6"):
		normalized = "low"
	case containsAny(severityValue, "debug", "trace", "7"):
		normalized = "debug"
	default:
		normalized = "low" // Default to low if unrecognized
	}

	event["severity"] = normalized
	event["_original_severity"] = severityValue
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
