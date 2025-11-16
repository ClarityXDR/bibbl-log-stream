package filters

import (
	"regexp"
	"strings"
	"testing"
)

func TestPIIRedactorSSN(t *testing.T) {
	pr := NewPIIRedactor()

	input := "My SSN is 123-45-6789 and my friend's is 987-65-4321"
	result := pr.Redact(input, "[REDACTED]")

	if strings.Contains(result, "123-45-6789") {
		t.Error("SSN was not redacted")
	}

	if strings.Contains(result, "987-65-4321") {
		t.Error("Second SSN was not redacted")
	}

	if !strings.Contains(result, "[REDACTED]") {
		t.Error("Replacement text not found")
	}
}

func TestPIIRedactorEmail(t *testing.T) {
	pr := NewPIIRedactor()

	input := "Contact us at support@example.com or admin@test.org"
	result := pr.Redact(input, "[EMAIL]")

	if strings.Contains(result, "support@example.com") {
		t.Error("Email was not redacted")
	}

	if strings.Contains(result, "admin@test.org") {
		t.Error("Second email was not redacted")
	}
}

func TestPIIRedactorCreditCard(t *testing.T) {
	pr := NewPIIRedactor()

	input := "Card number: 1234-5678-9012-3456"
	result := pr.Redact(input, "[CC]")

	if strings.Contains(result, "1234-5678-9012-3456") {
		t.Error("Credit card was not redacted")
	}
}

func TestPIIRedactorWithTags(t *testing.T) {
	pr := NewPIIRedactor()

	input := "SSN: 123-45-6789, Email: test@example.com"
	result := pr.RedactWithTags(input)

	if !strings.Contains(result, "[SSN]") {
		t.Error("SSN tag not found")
	}

	if !strings.Contains(result, "[EMAIL]") {
		t.Error("Email tag not found")
	}

	if strings.Contains(result, "123-45-6789") || strings.Contains(result, "test@example.com") {
		t.Error("Original PII still present")
	}
}

func TestPIIDetector(t *testing.T) {
	pr := NewPIIRedactor()

	input := "SSN: 123-45-6789, Email: test@example.com, Phone: 555-123-4567"
	detected := pr.DetectPII(input)

	if detected["ssn"] != 1 {
		t.Errorf("Expected 1 SSN, got %d", detected["ssn"])
	}

	if detected["email"] != 1 {
		t.Errorf("Expected 1 email, got %d", detected["email"])
	}

	if detected["phone"] != 1 {
		t.Errorf("Expected 1 phone, got %d", detected["phone"])
	}
}

func TestPIIRedactorMap(t *testing.T) {
	pr := NewPIIRedactor()

	input := map[string]interface{}{
		"message": "My SSN is 123-45-6789",
		"user": map[string]interface{}{
			"email": "test@example.com",
			"phone": "555-123-4567",
		},
		"cards": []interface{}{
			"1234-5678-9012-3456",
			"4321-8765-2109-6543",
		},
	}

	result := pr.RedactMap(input, "[REDACTED]")

	// Check top-level string
	if msg, ok := result["message"].(string); ok {
		if strings.Contains(msg, "123-45-6789") {
			t.Error("SSN in message was not redacted")
		}
	}

	// Check nested map
	if user, ok := result["user"].(map[string]interface{}); ok {
		if email, ok := user["email"].(string); ok {
			if strings.Contains(email, "test@example.com") {
				t.Error("Email was not redacted")
			}
		}
	}

	// Check array
	if cards, ok := result["cards"].([]interface{}); ok {
		for _, card := range cards {
			if cardStr, ok := card.(string); ok {
				if strings.Contains(cardStr, "1234-5678") || strings.Contains(cardStr, "4321-8765") {
					t.Error("Credit card in array was not redacted")
				}
			}
		}
	}
}

func TestCustomPattern(t *testing.T) {
	pr := NewPIIRedactor()

	// Add custom pattern for account numbers
	pr.AddPattern("account", regexp.MustCompile(`\bACCT-\d{8}\b`))

	input := "Account: ACCT-12345678"
	result := pr.Redact(input, "[REDACTED]")

	if strings.Contains(result, "ACCT-12345678") {
		t.Error("Custom account pattern was not redacted")
	}
}

func BenchmarkPIIRedact(b *testing.B) {
	pr := NewPIIRedactor()
	input := "Contact: test@example.com, SSN: 123-45-6789, Phone: 555-123-4567, Card: 1234-5678-9012-3456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pr.Redact(input, "[REDACTED]")
	}
}

func BenchmarkPIIDetect(b *testing.B) {
	pr := NewPIIRedactor()
	input := "Contact: test@example.com, SSN: 123-45-6789, Phone: 555-123-4567"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pr.DetectPII(input)
	}
}
