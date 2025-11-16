package filters

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// VersaKVPParser parses Versa Networks syslog messages in KVP (Key-Value Pair) format.
// Format: space-delimited key=value pairs with optional quoted values
// Example: 2017-11-26T22:42:38+0000 flowIdLog, applianceName=Branch1, tenantName=Customer1, flowId=33655871
//
// Supports all Versa log types:
// - flowIdLog, flowMonLog, accessLog, idpLog, urlfLog, avLog
// - cgnatLog, adcL4Log, authEventLog, casbLog, dlpLog, dnsfLog
// - fileFilterLog, saseWebLog, sandboxLog, and many more
type VersaKVPParser struct {
	// PreserveRaw ensures the original _raw field is kept
	PreserveRaw bool
	// StrictMode returns errors on parsing failures (default: false, skip malformed KVPs)
	StrictMode bool
}

// NewVersaKVPParser creates a new Versa KVP parser with sensible defaults
func NewVersaKVPParser() *VersaKVPParser {
	return &VersaKVPParser{
		PreserveRaw: true,  // Always preserve _raw for legal compliance
		StrictMode:  false, // Be lenient with malformed data
	}
}

// Parse extracts key-value pairs from Versa syslog KVP format
func (p *VersaKVPParser) Parse(event map[string]interface{}) error {
	// Get the raw syslog message
	raw, ok := event["_raw"]
	if !ok {
		return fmt.Errorf("no _raw field in event")
	}

	rawStr, ok := raw.(string)
	if !ok {
		return fmt.Errorf("_raw field is not a string")
	}

	// Parse the message
	parsed, err := p.parseKVP(rawStr)
	if err != nil && p.StrictMode {
		return fmt.Errorf("failed to parse KVP: %w", err)
	}

	// Merge parsed fields into event (preserving _raw)
	for k, v := range parsed {
		event[k] = v
	}

	// Always preserve _raw for legal/compliance requirements
	if p.PreserveRaw {
		event["_raw"] = rawStr
	}

	// Add metadata
	event["_parser"] = "versa_kvp"
	event["_parsed_at"] = time.Now().UTC().Format(time.RFC3339)

	// Extract log type from first KVP after timestamp
	if logType, ok := parsed["_log_type"].(string); ok {
		event["versa_log_type"] = logType
	}

	return nil
}

// parseKVP parses Versa KVP format: key=value key2=value2
// Handles quoted values, equals signs in values, and special characters
func (p *VersaKVPParser) parseKVP(raw string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Versa format: TIMESTAMP LOGTYPE, key1=value1, key2=value2, ...
	// Example: 2017-11-26T22:42:38+0000 flowIdLog, applianceName=Branch1, tenantName=Customer1

	// Step 1: Extract timestamp (combined date+time+zone format)
	// Format: 2017-11-26T22:42:38+0000
	firstSpace := strings.Index(raw, " ")
	if firstSpace == -1 {
		return result, fmt.Errorf("invalid Versa log format: no timestamp found")
	}

	timestamp := raw[:firstSpace]
	result["@timestamp"] = timestamp

	// Parse timestamp to ISO8601
	if ts, err := time.Parse("2006-01-02T15:04:05-0700", timestamp); err == nil {
		result["@timestamp_parsed"] = ts.UTC().Format(time.RFC3339Nano)
	}

	// Step 2: Extract log type (everything between first space and first comma)
	remainder := strings.TrimSpace(raw[firstSpace+1:])
	commaIdx := strings.Index(remainder, ",")
	if commaIdx == -1 {
		return result, fmt.Errorf("invalid Versa log format: no comma after log type")
	}

	logType := strings.TrimSpace(remainder[:commaIdx])
	result["_log_type"] = logType
	remainder = strings.TrimSpace(remainder[commaIdx+1:]) // Skip comma and whitespace

	// Step 3: Parse KVP pairs
	kvps := p.splitKVPs(remainder)
	for _, kvp := range kvps {
		key, value := p.parseKeyValue(kvp)
		if key != "" {
			// Type conversion for common numeric fields
			if typedValue := p.convertType(key, value); typedValue != nil {
				result[key] = typedValue
			} else {
				result[key] = value
			}
		}
	}

	return result, nil
}

// splitKVPs splits the KVP string on commas, respecting quoted values
func (p *VersaKVPParser) splitKVPs(s string) []string {
	var result []string
	var current strings.Builder
	inQuotes := false
	escape := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if escape {
			current.WriteByte(ch)
			escape = false
			continue
		}

		switch ch {
		case '\\':
			escape = true
			current.WriteByte(ch)
		case '"':
			inQuotes = !inQuotes
			current.WriteByte(ch)
		case ',':
			if !inQuotes {
				// End of KVP
				if kvp := strings.TrimSpace(current.String()); kvp != "" {
					result = append(result, kvp)
				}
				current.Reset()
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}

	// Add last KVP
	if kvp := strings.TrimSpace(current.String()); kvp != "" {
		result = append(result, kvp)
	}

	return result
}

// parseKeyValue splits a single KVP into key and value
func (p *VersaKVPParser) parseKeyValue(kvp string) (string, string) {
	// Find first equals sign
	idx := strings.Index(kvp, "=")
	if idx == -1 {
		return "", ""
	}

	key := strings.TrimSpace(kvp[:idx])
	value := strings.TrimSpace(kvp[idx+1:])

	// Remove quotes from value if present
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
		// Unescape quotes inside
		value = strings.ReplaceAll(value, `\"`, `"`)
		value = strings.ReplaceAll(value, `\\`, `\`)
	}

	return key, value
}

// convertType attempts to convert string values to appropriate types
func (p *VersaKVPParser) convertType(key, value string) interface{} {
	if value == "" {
		return ""
	}

	// Known integer fields
	intFields := map[string]bool{
		"flowId": true, "flowCookie": true, "tenantId": true, "applianceId": true,
		"vsnId": true, "sourcePort": true, "destinationPort": true,
		"sourceTransportPort": true, "destinationTransportPort": true,
		"protocolIdentifier": true, "sentOctets": true, "sentPackets": true,
		"recvdOctets": true, "recvdPackets": true, "fileSize": true,
		"appId": true, "signatureId": true, "groupId": true, "moduleId": true,
		"HitCount": true, "appRisk": true, "appProductivity": true,
		"flowStartMilliseconds": true, "flowEndMilliseconds": true,
		"observationTimeMilliseconds": true,
	}

	// Known float fields
	floatFields := map[string]bool{
		"latency": true, "jitter": true, "loss": true,
	}

	if intFields[key] {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}

	if floatFields[key] {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}

	// Return as string
	return value
}

// GetCommonFields returns a list of common Versa log fields for documentation
func GetCommonVersaFields() []string {
	return []string{
		// Universal fields
		"applianceName", "tenantName", "tenantId", "flowId", "flowCookie",
		"vsnId", "applianceId", "sourceIPv4Address", "sourceIPv6Address",
		"destinationIPv4Address", "destinationIPv6Address",
		"sourceTransportPort", "destinationTransportPort", "protocolIdentifier",

		// Flow fields
		"ingressInterfaceName", "egressInterfaceName", "fromZone", "toZone",
		"fromCountry", "toCountry", "fromUser", "sentOctets", "sentPackets",
		"recvdOctets", "recvdPackets", "flowStartMilliseconds", "flowEndMilliseconds",

		// Security fields
		"action", "rule", "profileName", "threatType", "threatSeverity",
		"urlCategory", "urlReputation", "appIdStr", "appFamily", "appSubFamily",
		"appRisk", "appProductivity",

		// IDP fields
		"signatureId", "signatureMsg", "idpAction", "ipsProfile", "ipsProfileRule",

		// URL filtering
		"urlfAction", "httpUrl", "httpHost", "httpMethod",

		// Antivirus
		"avAction", "avMalwareName", "fileName", "fileType", "fileTransDir",

		// DNS
		"dnsfAction", "dnsfDomain", "dnsfEvType", "dnsResponseCode",

		// CGNAT
		"postNATSourceIPv4Address", "postNATDestinationIPv4Address",
		"natRuleName", "natEvent",
	}
}
