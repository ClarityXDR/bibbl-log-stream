package filters

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// PaloAltoCSVParser parses Palo Alto Networks NGFW syslog messages in CSV format.
// Format: Comma-separated values with specific field positions per log type
//
// Supports all Palo Alto log types:
// - TRAFFIC: Network traffic logs (start, end, drop, deny)
// - THREAT: Vulnerability, spyware, virus, wildfire, url-filtering
// - CONFIG: Configuration changes
// - SYSTEM: System events
// - AUTHENTICATION: User authentication events
// - USERID: User-ID mapping events
// - HIP-MATCH: Host Information Profile matching
// - GLOBALPROTECT: GlobalProtect VPN logs
// - DECRYPTION: SSL/TLS decryption logs
// - TUNNEL: Tunnel inspection logs
// - SCTP: SCTP protocol logs
// - CORRELATION: Correlated events
// - GTP: GPRS Tunneling Protocol logs
// - AUDIT: Administrative audit logs
type PaloAltoCSVParser struct {
	// PreserveRaw ensures the original _raw field is kept
	PreserveRaw bool
	// StrictMode returns errors on parsing failures (default: false, lenient)
	StrictMode bool
}

// NewPaloAltoCSVParser creates a new Palo Alto CSV parser with sensible defaults
func NewPaloAltoCSVParser() *PaloAltoCSVParser {
	return &PaloAltoCSVParser{
		PreserveRaw: true,  // Always preserve _raw for legal compliance
		StrictMode:  false, // Be lenient with malformed data
	}
}

// Parse extracts CSV fields from Palo Alto syslog format
func (p *PaloAltoCSVParser) Parse(event map[string]interface{}) error {
	// Get the raw syslog message
	raw, ok := event["_raw"]
	if !ok {
		return fmt.Errorf("no _raw field in event")
	}

	rawStr, ok := raw.(string)
	if !ok {
		return fmt.Errorf("_raw field is not a string")
	}

	// Parse the CSV message
	parsed, err := p.parseCSV(rawStr)
	if err != nil && p.StrictMode {
		return fmt.Errorf("failed to parse CSV: %w", err)
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
	event["_parser"] = "paloalto_csv"
	event["_parsed_at"] = time.Now().UTC().Format(time.RFC3339)

	// Extract log type
	if logType, ok := parsed["type"].(string); ok {
		event["paloalto_log_type"] = logType
	}

	return nil
}

// parseCSV parses Palo Alto CSV format
func (p *PaloAltoCSVParser) parseCSV(raw string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Use CSV reader to handle quoted fields and escaped commas
	reader := csv.NewReader(strings.NewReader(raw))
	reader.LazyQuotes = true // Be lenient with quotes
	reader.TrimLeadingSpace = true

	// Read all fields
	fields, err := reader.Read()
	if err != nil {
		return result, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(fields) < 10 {
		return result, fmt.Errorf("insufficient fields: got %d, need at least 10", len(fields))
	}

	// Common fields present in all log types (first ~10 fields)
	// Field positions: 0=FUTURE_USE, 1=Receive Time, 2=Serial, 3=Type, 4=Subtype, etc.
	p.parseCommonFields(fields, result)

	// Parse type-specific fields based on log type
	logType := getString(fields, 3)
	result["type"] = logType

	switch strings.ToUpper(logType) {
	case "TRAFFIC":
		p.parseTrafficFields(fields, result)
	case "THREAT":
		p.parseThreatFields(fields, result)
	case "CONFIG":
		p.parseConfigFields(fields, result)
	case "SYSTEM":
		p.parseSystemFields(fields, result)
	case "AUTHENTICATION":
		p.parseAuthenticationFields(fields, result)
	case "USERID":
		p.parseUserIDFields(fields, result)
	case "HIP-MATCH", "HIPMATCH":
		p.parseHIPMatchFields(fields, result)
	case "GLOBALPROTECT":
		p.parseGlobalProtectFields(fields, result)
	case "DECRYPTION":
		p.parseDecryptionFields(fields, result)
	case "TUNNEL":
		p.parseTunnelFields(fields, result)
	case "SCTP":
		p.parseSCTPFields(fields, result)
	case "CORRELATION":
		p.parseCorrelationFields(fields, result)
	case "GTP":
		p.parseGTPFields(fields, result)
	case "AUDIT":
		p.parseAuditFields(fields, result)
	default:
		// Unknown log type - parse generic fields
		p.parseGenericFields(fields, result)
	}

	return result, nil
}

// parseCommonFields extracts fields common to all log types
func (p *PaloAltoCSVParser) parseCommonFields(fields []string, result map[string]interface{}) {
	// Field 1: Receive Time
	if receiveTime := getString(fields, 1); receiveTime != "" {
		result["receive_time"] = receiveTime
		// Try to parse timestamp
		if ts, err := time.Parse("2006/01/02 15:04:05", receiveTime); err == nil {
			result["@timestamp"] = ts.UTC().Format(time.RFC3339Nano)
		}
	}

	// Field 2: Serial Number
	if serial := getString(fields, 2); serial != "" {
		result["serial"] = serial
	}

	// Field 3: Type (already set by caller)
	// Field 4: Subtype/Threat Content Type
	if subtype := getString(fields, 4); subtype != "" {
		result["subtype"] = subtype
	}

	// Field 6: Generated Time
	if genTime := getString(fields, 6); genTime != "" {
		result["time_generated"] = genTime
	}
}

// parseTrafficFields extracts TRAFFIC log specific fields
func (p *PaloAltoCSVParser) parseTrafficFields(fields []string, result map[string]interface{}) {
	// Traffic logs have ~100+ fields
	// Key fields: src, dst, natsrc, natdst, rule, srcuser, dstuser, app, etc.

	result["src"] = getString(fields, 7)                 // Source Address
	result["dst"] = getString(fields, 8)                 // Destination Address
	result["natsrc"] = getString(fields, 9)              // NAT Source IP
	result["natdst"] = getString(fields, 10)             // NAT Destination IP
	result["rule"] = getString(fields, 11)               // Rule Name
	result["srcuser"] = getString(fields, 12)            // Source User
	result["dstuser"] = getString(fields, 13)            // Destination User
	result["app"] = getString(fields, 14)                // Application
	result["vsys"] = getString(fields, 15)               // Virtual System
	result["from"] = getString(fields, 16)               // Source Zone
	result["to"] = getString(fields, 17)                 // Destination Zone
	result["inbound_if"] = getString(fields, 18)         // Inbound Interface
	result["outbound_if"] = getString(fields, 19)        // Outbound Interface
	result["logset"] = getString(fields, 20)             // Log Action
	result["sessionid"] = getInt(fields, 22)             // Session ID
	result["repeatcnt"] = getInt(fields, 23)             // Repeat Count
	result["sport"] = getInt(fields, 24)                 // Source Port
	result["dport"] = getInt(fields, 25)                 // Destination Port
	result["natsport"] = getInt(fields, 26)              // NAT Source Port
	result["natdport"] = getInt(fields, 27)              // NAT Destination Port
	result["flags"] = getString(fields, 28)              // Flags
	result["proto"] = getString(fields, 29)              // IP Protocol
	result["action"] = getString(fields, 30)             // Action
	result["bytes"] = getInt(fields, 31)                 // Total Bytes
	result["bytes_sent"] = getInt(fields, 32)            // Bytes Sent
	result["bytes_received"] = getInt(fields, 33)        // Bytes Received
	result["packets"] = getInt(fields, 34)               // Total Packets
	result["start"] = getString(fields, 35)              // Start Time
	result["elapsed"] = getInt(fields, 36)               // Elapsed Time
	result["category"] = getString(fields, 37)           // Category
	result["seqno"] = getInt(fields, 39)                 // Sequence Number
	result["actionflags"] = getString(fields, 40)        // Action Flags
	result["srcloc"] = getString(fields, 41)             // Source Country
	result["dstloc"] = getString(fields, 42)             // Destination Country
	result["pkts_sent"] = getInt(fields, 44)             // Packets Sent
	result["pkts_received"] = getInt(fields, 45)         // Packets Received
	result["session_end_reason"] = getString(fields, 46) // Session End Reason

	// Device Group Hierarchy
	if len(fields) > 47 {
		result["dg_hier_level_1"] = getInt(fields, 47)
		result["dg_hier_level_2"] = getInt(fields, 48)
		result["dg_hier_level_3"] = getInt(fields, 49)
		result["dg_hier_level_4"] = getInt(fields, 50)
	}

	// Additional fields (51+)
	if len(fields) > 51 {
		result["vsys_name"] = getString(fields, 51)   // Virtual System Name
		result["device_name"] = getString(fields, 52) // Device Name
		result["action_source"] = getString(fields, 53)
		result["src_uuid"] = getString(fields, 54)
		result["dst_uuid"] = getString(fields, 55)
		result["tunnelid"] = getString(fields, 56)
		result["imsi"] = getString(fields, 56) // Same field
		result["monitortag"] = getString(fields, 57)
		result["imei"] = getString(fields, 57) // Same field
		result["parent_session_id"] = getString(fields, 58)
		result["parent_start_time"] = getString(fields, 59)
		result["tunnel"] = getString(fields, 60)
	}

	// SCTP fields (if present)
	if len(fields) > 61 {
		result["assoc_id"] = getInt(fields, 61)
		result["chunks"] = getInt(fields, 62)
		result["chunks_sent"] = getInt(fields, 63)
		result["chunks_received"] = getInt(fields, 64)
		result["rule_uuid"] = getString(fields, 65)
		result["http2_connection"] = getString(fields, 66)
	}

	// Extended fields (67+)
	if len(fields) > 67 {
		result["link_change_count"] = getInt(fields, 67)
		result["policy_id"] = getString(fields, 68)
		result["link_switches"] = getString(fields, 69)
		result["sdwan_cluster"] = getString(fields, 70)
		result["sdwan_device_type"] = getString(fields, 71)
		result["sdwan_cluster_type"] = getString(fields, 72)
		result["sdwan_site"] = getString(fields, 73)
		result["dynusergroup_name"] = getString(fields, 74)
		result["xff_ip"] = getString(fields, 75)
	}

	// Device-ID fields (76+)
	if len(fields) > 76 {
		result["src_category"] = getString(fields, 76)
		result["src_profile"] = getString(fields, 77)
		result["src_model"] = getString(fields, 78)
		result["src_vendor"] = getString(fields, 79)
		result["src_osfamily"] = getString(fields, 80)
		result["src_osversion"] = getString(fields, 81)
		result["src_host"] = getString(fields, 82)
		result["src_mac"] = getString(fields, 83)
	}

	if len(fields) > 84 {
		result["dst_category"] = getString(fields, 84)
		result["dst_profile"] = getString(fields, 85)
		result["dst_model"] = getString(fields, 86)
		result["dst_vendor"] = getString(fields, 87)
		result["dst_osfamily"] = getString(fields, 88)
		result["dst_osversion"] = getString(fields, 89)
		result["dst_host"] = getString(fields, 90)
		result["dst_mac"] = getString(fields, 91)
	}

	// Container/Kubernetes fields (92+)
	if len(fields) > 92 {
		result["container_id"] = getString(fields, 92)
		result["pod_namespace"] = getString(fields, 93)
		result["pod_name"] = getString(fields, 94)
		result["src_edl"] = getString(fields, 95)
		result["dst_edl"] = getString(fields, 96)
		result["hostid"] = getString(fields, 97)
		result["serialnumber"] = getString(fields, 98)
		result["src_dag"] = getString(fields, 99)
		result["dst_dag"] = getString(fields, 100)
		result["session_owner"] = getString(fields, 101)
		result["high_res_timestamp"] = getString(fields, 102)
	}

	// 5G/Network Slice fields (103+)
	if len(fields) > 103 {
		result["nssai_sst"] = getString(fields, 103)
		result["nssai_sd"] = getString(fields, 104)
	}

	// Application metadata fields (105+)
	if len(fields) > 105 {
		result["subcategory_of_app"] = getString(fields, 105)
		result["category_of_app"] = getString(fields, 106)
		result["technology_of_app"] = getString(fields, 107)
		result["risk_of_app"] = getInt(fields, 108)
		result["characteristic_of_app"] = getString(fields, 109)
		result["container_of_app"] = getString(fields, 110)
		result["tunneled_app"] = getString(fields, 111)
		result["is_saas_of_app"] = getInt(fields, 112)
		result["sanctioned_state_of_app"] = getInt(fields, 113)
	}

	// Additional fields (114+)
	if len(fields) > 114 {
		result["offloaded"] = getInt(fields, 114)
		result["flow_type"] = getString(fields, 115)
		result["cluster_name"] = getString(fields, 116)
	}
}

// parseThreatFields extracts THREAT log specific fields
func (p *PaloAltoCSVParser) parseThreatFields(fields []string, result map[string]interface{}) {
	// Threat logs include vulnerability, spyware, virus, wildfire, url-filtering, data-filtering
	result["src"] = getString(fields, 7)
	result["dst"] = getString(fields, 8)
	result["natsrc"] = getString(fields, 9)
	result["natdst"] = getString(fields, 10)
	result["rule"] = getString(fields, 11)
	result["srcuser"] = getString(fields, 12)
	result["dstuser"] = getString(fields, 13)
	result["app"] = getString(fields, 14)
	result["vsys"] = getString(fields, 15)
	result["from"] = getString(fields, 16)
	result["to"] = getString(fields, 17)
	result["inbound_if"] = getString(fields, 18)
	result["outbound_if"] = getString(fields, 19)
	result["logset"] = getString(fields, 20)
	result["sessionid"] = getInt(fields, 22)
	result["repeatcnt"] = getInt(fields, 23)
	result["sport"] = getInt(fields, 24)
	result["dport"] = getInt(fields, 25)
	result["natsport"] = getInt(fields, 26)
	result["natdport"] = getInt(fields, 27)
	result["flags"] = getString(fields, 28)
	result["proto"] = getString(fields, 29)
	result["action"] = getString(fields, 30)

	// Threat-specific fields
	if len(fields) > 31 {
		result["misc"] = getString(fields, 31) // Threat/Filename/URL
		result["threatid"] = getString(fields, 32)
		result["category"] = getString(fields, 33)
		result["severity"] = getString(fields, 34)
		result["direction"] = getString(fields, 35)
		result["seqno"] = getInt(fields, 36)
		result["actionflags"] = getString(fields, 37)
		result["srcloc"] = getString(fields, 38)
		result["dstloc"] = getString(fields, 39)
		result["contenttype"] = getString(fields, 41)
	}

	// Additional threat fields
	if len(fields) > 42 {
		result["pcap_id"] = getString(fields, 42)
		result["filedigest"] = getString(fields, 43)
		result["cloud"] = getString(fields, 44)
		result["url_idx"] = getInt(fields, 45)
		result["user_agent"] = getString(fields, 46)
		result["filetype"] = getString(fields, 47)
		result["xff"] = getString(fields, 48)
		result["referer"] = getString(fields, 49)
		result["sender"] = getString(fields, 50)
		result["subject"] = getString(fields, 51)
		result["recipient"] = getString(fields, 52)
		result["reportid"] = getString(fields, 53)
	}

	// Device Group Hierarchy
	if len(fields) > 54 {
		result["dg_hier_level_1"] = getInt(fields, 54)
		result["dg_hier_level_2"] = getInt(fields, 55)
		result["dg_hier_level_3"] = getInt(fields, 56)
		result["dg_hier_level_4"] = getInt(fields, 57)
		result["vsys_name"] = getString(fields, 58)
		result["device_name"] = getString(fields, 59)
	}

	// Extended threat fields
	if len(fields) > 60 {
		result["src_uuid"] = getString(fields, 60)
		result["dst_uuid"] = getString(fields, 61)
		result["http_method"] = getString(fields, 62)
		result["tunnel_id"] = getString(fields, 63)
		result["imsi"] = getString(fields, 63)
		result["monitortag"] = getString(fields, 64)
		result["imei"] = getString(fields, 64)
		result["parent_session_id"] = getString(fields, 65)
		result["parent_start_time"] = getString(fields, 66)
		result["tunnel"] = getString(fields, 67)
		result["thr_category"] = getString(fields, 68)
		result["contentver"] = getString(fields, 69)
	}

	// More extended fields (70+)
	if len(fields) > 70 {
		result["sig_flags"] = getString(fields, 70)
		result["rule_uuid"] = getString(fields, 71)
		result["http2_connection"] = getString(fields, 72)
		result["dynusergroup_name"] = getString(fields, 73)
		result["xff_ip"] = getString(fields, 74)
		result["src_category"] = getString(fields, 75)
		result["src_profile"] = getString(fields, 76)
		result["src_model"] = getString(fields, 77)
		result["src_vendor"] = getString(fields, 78)
		result["src_osfamily"] = getString(fields, 79)
		result["src_osversion"] = getString(fields, 80)
		result["src_host"] = getString(fields, 81)
		result["src_mac"] = getString(fields, 82)
	}

	// Destination device fields
	if len(fields) > 83 {
		result["dst_category"] = getString(fields, 83)
		result["dst_profile"] = getString(fields, 84)
		result["dst_model"] = getString(fields, 85)
		result["dst_vendor"] = getString(fields, 86)
		result["dst_osfamily"] = getString(fields, 87)
		result["dst_osversion"] = getString(fields, 88)
		result["dst_host"] = getString(fields, 89)
		result["dst_mac"] = getString(fields, 90)
	}

	// Container/K8s fields
	if len(fields) > 91 {
		result["container_id"] = getString(fields, 91)
		result["pod_namespace"] = getString(fields, 92)
		result["pod_name"] = getString(fields, 93)
		result["src_edl"] = getString(fields, 94)
		result["dst_edl"] = getString(fields, 95)
		result["hostid"] = getString(fields, 96)
		result["serialnumber"] = getString(fields, 97)
	}
}

// parseConfigFields extracts CONFIG log specific fields
func (p *PaloAltoCSVParser) parseConfigFields(fields []string, result map[string]interface{}) {
	result["host"] = getString(fields, 7)
	result["cmd"] = getString(fields, 8)
	result["admin"] = getString(fields, 9)
	result["client"] = getString(fields, 10)
	result["config_result"] = getString(fields, 11)
	result["path"] = getString(fields, 12)

	if len(fields) > 13 {
		result["before_change_detail"] = getString(fields, 13)
		result["after_change_detail"] = getString(fields, 14)
		result["seqno"] = getInt(fields, 15)
		result["actionflags"] = getString(fields, 16)
	}

	if len(fields) > 17 {
		result["dg_hier_level_1"] = getInt(fields, 17)
		result["dg_hier_level_2"] = getInt(fields, 18)
		result["dg_hier_level_3"] = getInt(fields, 19)
		result["dg_hier_level_4"] = getInt(fields, 20)
		result["vsys_name"] = getString(fields, 21)
		result["device_name"] = getString(fields, 22)
	}
}

// parseSystemFields extracts SYSTEM log specific fields
// Format: FUTURE_USE, Receive Time, Serial, Type, Subtype, FUTURE_USE, Generated Time,
// Virtual System, Event ID, Object, FUTURE_USE, FUTURE_USE, Module, Severity, Description, Seqno, ActionFlags...
func (p *PaloAltoCSVParser) parseSystemFields(fields []string, result map[string]interface{}) {
	result["vsys"] = getString(fields, 7)
	result["eventid"] = getString(fields, 8)
	result["object"] = getString(fields, 9)
	// Fields 10-11 are FUTURE_USE
	result["module"] = getString(fields, 12)
	result["severity"] = getString(fields, 13)
	result["opaque"] = getString(fields, 14)
	result["seqno"] = getInt(fields, 15)
	result["actionflags"] = getString(fields, 16)

	if len(fields) > 17 {
		result["dg_hier_level_1"] = getInt(fields, 17)
		result["dg_hier_level_2"] = getInt(fields, 18)
		result["dg_hier_level_3"] = getInt(fields, 19)
		result["dg_hier_level_4"] = getInt(fields, 20)
		result["vsys_name"] = getString(fields, 21)
		result["device_name"] = getString(fields, 22)
	}
}

// parseAuthenticationFields extracts AUTHENTICATION log fields
func (p *PaloAltoCSVParser) parseAuthenticationFields(fields []string, result map[string]interface{}) {
	result["vsys"] = getString(fields, 7)
	result["ip"] = getString(fields, 8)
	result["user"] = getString(fields, 9)
	result["normalize_user"] = getString(fields, 10)
	result["object"] = getString(fields, 11)
	result["authpolicy"] = getString(fields, 12)
	result["repeatcnt"] = getInt(fields, 13)
	result["authid"] = getString(fields, 14)
	result["vendor"] = getString(fields, 15)
	result["logset"] = getString(fields, 16)
	result["serverprofile"] = getString(fields, 17)
	result["description"] = getString(fields, 18)
	result["clienttype"] = getString(fields, 19)
	result["event"] = getString(fields, 20)
	result["factorno"] = getInt(fields, 21)
	result["seqno"] = getInt(fields, 22)
	result["actionflags"] = getString(fields, 23)

	if len(fields) > 24 {
		result["dg_hier_level_1"] = getInt(fields, 24)
		result["dg_hier_level_2"] = getInt(fields, 25)
		result["dg_hier_level_3"] = getInt(fields, 26)
		result["dg_hier_level_4"] = getInt(fields, 27)
		result["vsys_name"] = getString(fields, 28)
		result["device_name"] = getString(fields, 29)
		result["vsys_id"] = getInt(fields, 30)
		result["auth_protocol"] = getString(fields, 31)
	}
}

// parseUserIDFields extracts USERID log fields
func (p *PaloAltoCSVParser) parseUserIDFields(fields []string, result map[string]interface{}) {
	result["vsys"] = getString(fields, 7)
	result["ip"] = getString(fields, 8)
	result["user"] = getString(fields, 9)
	result["datasourcename"] = getString(fields, 10)
	result["eventid"] = getString(fields, 11)
	result["repeatcnt"] = getInt(fields, 12)
	result["timeout"] = getInt(fields, 13)
	result["beginport"] = getInt(fields, 14)
	result["endport"] = getInt(fields, 15)
	result["datasource"] = getString(fields, 16)
	result["datasourcetype"] = getString(fields, 17)
	result["seqno"] = getInt(fields, 18)
	result["actionflags"] = getString(fields, 19)

	if len(fields) > 20 {
		result["dg_hier_level_1"] = getInt(fields, 20)
		result["dg_hier_level_2"] = getInt(fields, 21)
		result["dg_hier_level_3"] = getInt(fields, 22)
		result["dg_hier_level_4"] = getInt(fields, 23)
		result["vsys_name"] = getString(fields, 24)
		result["device_name"] = getString(fields, 25)
		result["vsys_id"] = getInt(fields, 26)
		result["factortype"] = getString(fields, 27)
		result["factorcompletiontime"] = getString(fields, 28)
	}
}

// parseHIPMatchFields extracts HIP-MATCH log fields
func (p *PaloAltoCSVParser) parseHIPMatchFields(fields []string, result map[string]interface{}) {
	result["srcuser"] = getString(fields, 7)
	result["vsys"] = getString(fields, 8)
	result["machinename"] = getString(fields, 9)
	result["os"] = getString(fields, 10)
	result["src"] = getString(fields, 11)
	result["matchname"] = getString(fields, 12)
	result["repeatcnt"] = getInt(fields, 13)
	result["matchtype"] = getString(fields, 14)
	result["seqno"] = getInt(fields, 15)
	result["actionflags"] = getString(fields, 16)

	if len(fields) > 17 {
		result["dg_hier_level_1"] = getInt(fields, 17)
		result["dg_hier_level_2"] = getInt(fields, 18)
		result["dg_hier_level_3"] = getInt(fields, 19)
		result["dg_hier_level_4"] = getInt(fields, 20)
		result["vsys_name"] = getString(fields, 21)
		result["device_name"] = getString(fields, 22)
		result["vsys_id"] = getInt(fields, 23)
		result["ipv6"] = getString(fields, 24)
		result["hostid"] = getString(fields, 25)
		result["serialnumber"] = getString(fields, 26)
	}
}

// parseGlobalProtectFields extracts GLOBALPROTECT log fields
func (p *PaloAltoCSVParser) parseGlobalProtectFields(fields []string, result map[string]interface{}) {
	result["vsys"] = getString(fields, 7)
	result["eventid"] = getString(fields, 8)
	result["stage"] = getString(fields, 9)
	result["auth_method"] = getString(fields, 10)
	result["tunnel_type"] = getString(fields, 11)
	result["srcuser"] = getString(fields, 12)
	result["srcregion"] = getString(fields, 13)
	result["machinename"] = getString(fields, 14)
	result["public_ip"] = getString(fields, 15)
	result["public_ipv6"] = getString(fields, 16)
	result["private_ip"] = getString(fields, 17)
	result["private_ipv6"] = getString(fields, 18)
	result["hostid"] = getString(fields, 19)
	result["serialnumber"] = getString(fields, 20)
	result["client_ver"] = getString(fields, 21)
	result["client_os"] = getString(fields, 22)
	result["client_os_ver"] = getString(fields, 23)
	result["repeatcnt"] = getInt(fields, 24)
	result["reason"] = getString(fields, 25)
	result["error"] = getString(fields, 26)
	result["opaque"] = getString(fields, 27)
	result["seqno"] = getInt(fields, 28)
	result["actionflags"] = getString(fields, 29)

	if len(fields) > 30 {
		result["event_time"] = getString(fields, 30)
		result["selection_type"] = getString(fields, 31)
		result["response_time"] = getInt(fields, 32)
		result["priority"] = getInt(fields, 33)
		result["attempted_gateways"] = getString(fields, 34)
		result["gateway"] = getString(fields, 35)
	}

	if len(fields) > 36 {
		result["dg_hier_level_1"] = getInt(fields, 36)
		result["dg_hier_level_2"] = getInt(fields, 37)
		result["dg_hier_level_3"] = getInt(fields, 38)
		result["dg_hier_level_4"] = getInt(fields, 39)
		result["vsys_name"] = getString(fields, 40)
		result["device_name"] = getString(fields, 41)
		result["vsys_id"] = getInt(fields, 42)
	}
}

// parseDecryptionFields extracts DECRYPTION log fields
func (p *PaloAltoCSVParser) parseDecryptionFields(fields []string, result map[string]interface{}) {
	result["src"] = getString(fields, 7)
	result["dst"] = getString(fields, 8)
	result["natsrc"] = getString(fields, 9)
	result["natdst"] = getString(fields, 10)
	result["rule"] = getString(fields, 11)
	result["srcuser"] = getString(fields, 12)
	result["dstuser"] = getString(fields, 13)
	result["app"] = getString(fields, 14)
	result["vsys"] = getString(fields, 15)
	result["from"] = getString(fields, 16)
	result["to"] = getString(fields, 17)
	result["inbound_if"] = getString(fields, 18)
	result["outbound_if"] = getString(fields, 19)
	result["logset"] = getString(fields, 20)
	result["sessionid"] = getInt(fields, 22)
	result["repeatcnt"] = getInt(fields, 23)
	result["sport"] = getInt(fields, 24)
	result["dport"] = getInt(fields, 25)
	result["natsport"] = getInt(fields, 26)
	result["natdport"] = getInt(fields, 27)
	result["flags"] = getString(fields, 28)
	result["proto"] = getString(fields, 29)
	result["action"] = getString(fields, 30)

	if len(fields) > 31 {
		result["policy_name"] = getString(fields, 31)
		result["decrypt_mirror"] = getString(fields, 32)
		result["ssl_version"] = getString(fields, 33)
		result["ssl_cipher_suite"] = getString(fields, 34)
		result["elliptic_curve"] = getString(fields, 35)
		result["error_index"] = getString(fields, 36)
		result["root_status"] = getString(fields, 37)
		result["chain_status"] = getString(fields, 38)
		result["proxy_type"] = getString(fields, 39)
		result["cert_serial"] = getString(fields, 40)
		result["fingerprint"] = getString(fields, 41)
		result["cert_start_time"] = getString(fields, 42)
		result["cert_end_time"] = getString(fields, 43)
		result["cert_version"] = getString(fields, 44)
		result["cert_size"] = getInt(fields, 45)
		result["cn_length"] = getInt(fields, 46)
		result["issuer_cn_length"] = getInt(fields, 47)
		result["root_cn_length"] = getInt(fields, 48)
	}
}

// parseTunnelFields extracts TUNNEL log fields
func (p *PaloAltoCSVParser) parseTunnelFields(fields []string, result map[string]interface{}) {
	result["src"] = getString(fields, 7)
	result["dst"] = getString(fields, 8)
	result["natsrc"] = getString(fields, 9)
	result["natdst"] = getString(fields, 10)
	result["rule"] = getString(fields, 11)
	result["srcuser"] = getString(fields, 12)
	result["dstuser"] = getString(fields, 13)
	result["app"] = getString(fields, 14)
	result["vsys"] = getString(fields, 15)
	result["from"] = getString(fields, 16)
	result["to"] = getString(fields, 17)
	result["inbound_if"] = getString(fields, 18)
	result["outbound_if"] = getString(fields, 19)
	result["logset"] = getString(fields, 20)
	result["sessionid"] = getInt(fields, 22)
	result["repeatcnt"] = getInt(fields, 23)
	result["sport"] = getInt(fields, 24)
	result["dport"] = getInt(fields, 25)
	result["natsport"] = getInt(fields, 26)
	result["natdport"] = getInt(fields, 27)
	result["flags"] = getString(fields, 28)
	result["proto"] = getString(fields, 29)
	result["action"] = getString(fields, 30)
}

// parseSCTPFields extracts SCTP log fields
func (p *PaloAltoCSVParser) parseSCTPFields(fields []string, result map[string]interface{}) {
	// SCTP logs are similar to traffic logs
	p.parseTrafficFields(fields, result)
}

// parseCorrelationFields extracts CORRELATION log fields
func (p *PaloAltoCSVParser) parseCorrelationFields(fields []string, result map[string]interface{}) {
	result["vsys"] = getString(fields, 7)
	result["category"] = getString(fields, 8)
	result["severity"] = getString(fields, 9)
	result["eventid"] = getString(fields, 10)
	result["object_name"] = getString(fields, 11)
	result["object_id"] = getString(fields, 12)
	result["evidence"] = getString(fields, 13)
	result["seqno"] = getInt(fields, 14)
	result["actionflags"] = getString(fields, 15)

	if len(fields) > 16 {
		result["dg_hier_level_1"] = getInt(fields, 16)
		result["dg_hier_level_2"] = getInt(fields, 17)
		result["dg_hier_level_3"] = getInt(fields, 18)
		result["dg_hier_level_4"] = getInt(fields, 19)
		result["vsys_name"] = getString(fields, 20)
		result["device_name"] = getString(fields, 21)
	}
}

// parseGTPFields extracts GTP log fields
func (p *PaloAltoCSVParser) parseGTPFields(fields []string, result map[string]interface{}) {
	result["src"] = getString(fields, 7)
	result["dst"] = getString(fields, 8)
	result["rule"] = getString(fields, 9)
	result["app"] = getString(fields, 10)
	result["vsys"] = getString(fields, 11)
	result["from"] = getString(fields, 12)
	result["to"] = getString(fields, 13)
	result["inbound_if"] = getString(fields, 14)
	result["outbound_if"] = getString(fields, 15)
	result["logset"] = getString(fields, 16)
	result["start"] = getString(fields, 18)
	result["sessionid"] = getInt(fields, 19)
	result["sport"] = getInt(fields, 20)
	result["dport"] = getInt(fields, 21)
	result["proto"] = getString(fields, 22)
	result["action"] = getString(fields, 23)
	result["event_type"] = getString(fields, 24)
	result["msisdn"] = getString(fields, 25)
	result["apn"] = getString(fields, 26)
	result["rat"] = getString(fields, 27)
	result["msg_type"] = getString(fields, 28)
	result["end_ip_addr"] = getString(fields, 29)
	result["teid1"] = getString(fields, 30)
	result["teid2"] = getString(fields, 31)
	result["gtp_interface"] = getString(fields, 32)
	result["cause_code"] = getInt(fields, 33)
	result["severity"] = getString(fields, 34)
	result["mcc"] = getString(fields, 35)
	result["mnc"] = getString(fields, 36)
	result["area_code"] = getInt(fields, 37)
	result["cell_id"] = getInt(fields, 38)
	result["event_code"] = getInt(fields, 39)
	result["srcloc"] = getString(fields, 40)
	result["dstloc"] = getString(fields, 41)
	result["imsi"] = getString(fields, 42)
	result["imei"] = getString(fields, 43)
	result["seqno"] = getInt(fields, 44)
	result["actionflags"] = getString(fields, 45)
}

// parseAuditFields extracts AUDIT log fields
func (p *PaloAltoCSVParser) parseAuditFields(fields []string, result map[string]interface{}) {
	result["before_change_detail"] = getString(fields, 7)
	result["after_change_detail"] = getString(fields, 8)
	result["audit_comment"] = getString(fields, 9)
	result["seqno"] = getInt(fields, 10)
	result["actionflags"] = getString(fields, 11)

	if len(fields) > 12 {
		result["dg_hier_level_1"] = getInt(fields, 12)
		result["dg_hier_level_2"] = getInt(fields, 13)
		result["dg_hier_level_3"] = getInt(fields, 14)
		result["dg_hier_level_4"] = getInt(fields, 15)
		result["vsys_name"] = getString(fields, 16)
		result["device_name"] = getString(fields, 17)
	}
}

// parseGenericFields extracts fields when log type is unknown
func (p *PaloAltoCSVParser) parseGenericFields(fields []string, result map[string]interface{}) {
	// Store all fields as field_N for unknown log types
	for i, field := range fields {
		if field != "" && i > 6 { // Skip first 7 common fields
			result[fmt.Sprintf("field_%d", i)] = field
		}
	}
}

// Helper functions for safe field extraction

func getString(fields []string, index int) string {
	if index >= 0 && index < len(fields) {
		return strings.TrimSpace(fields[index])
	}
	return ""
}

func getInt(fields []string, index int) interface{} {
	val := getString(fields, index)
	if val == "" {
		return nil
	}
	if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
		return intVal
	}
	return val // Return as string if not parseable as int
}

// GetCommonPaloAltoFields returns a list of common Palo Alto log fields
func GetCommonPaloAltoFields() []string {
	return []string{
		// Common fields
		"receive_time", "serial", "type", "subtype", "time_generated",

		// Network fields
		"src", "dst", "natsrc", "natdst", "sport", "dport",
		"natsport", "natdport", "proto",

		// Policy fields
		"rule", "action", "app", "vsys", "from", "to",
		"inbound_if", "outbound_if",

		// User fields
		"srcuser", "dstuser",

		// Session fields
		"sessionid", "bytes", "bytes_sent", "bytes_received",
		"packets", "pkts_sent", "pkts_received",

		// Metadata
		"category", "severity", "device_name", "vsys_name",
		"seqno", "actionflags",

		// Geography
		"srcloc", "dstloc",

		// Threat-specific
		"threatid", "misc", "direction", "filedigest",

		// Device-ID
		"src_category", "src_vendor", "src_model", "src_osfamily",
		"dst_category", "dst_vendor", "dst_model", "dst_osfamily",
	}
}
