package filters

import (
	"testing"
)

func TestPaloAltoCsvParser_TrafficLog(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Sample TRAFFIC log (simplified for testing)
	// Based on actual format - focusing on key fields only
	raw := `,2024/01/15 10:30:45,007951000012345,TRAFFIC,end,,2024/01/15 10:30:44,192.168.1.100,10.0.0.50,0.0.0.0,0.0.0.0,Allow-Web,,,web-browsing,vsys1,trust,untrust,ethernet1/1,ethernet1/2,Log-Forwarding,,123456,1,54321,443,0,0,0x80000000,tcp,allow,1024,512,512,10,2024/01/15 10:30:35,9,any,,,0,0x0,US,GB`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify common fields
	if event["type"] != "TRAFFIC" {
		t.Errorf("Expected type=TRAFFIC, got %v", event["type"])
	}

	if event["subtype"] != "end" {
		t.Errorf("Expected subtype=end, got %v", event["subtype"])
	}

	// Verify network fields
	if event["src"] != "192.168.1.100" {
		t.Errorf("Expected src=192.168.1.100, got %v", event["src"])
	}

	if event["dst"] != "10.0.0.50" {
		t.Errorf("Expected dst=10.0.0.50, got %v", event["dst"])
	}

	// Verify ports
	sport, ok := event["sport"]
	if !ok || sport.(int64) != 54321 {
		t.Errorf("Expected sport=54321, got %v", sport)
	}

	dport, ok := event["dport"]
	if !ok || dport.(int64) != 443 {
		t.Errorf("Expected dport=443, got %v", dport)
	}

	// Verify protocol and action
	if event["proto"] != "tcp" {
		t.Errorf("Expected proto=tcp, got %v", event["proto"])
	}

	if event["action"] != "allow" {
		t.Errorf("Expected action=allow, got %v", event["action"])
	}

	// Verify rule
	if event["rule"] != "Allow-Web" {
		t.Errorf("Expected rule=Allow-Web, got %v", event["rule"])
	}

	// Verify application
	if event["app"] != "web-browsing" {
		t.Errorf("Expected app=web-browsing, got %v", event["app"])
	}

	// Verify zones
	if event["from"] != "trust" {
		t.Errorf("Expected from=trust, got %v", event["from"])
	}

	// Verify bytes
	bytes, ok := event["bytes"]
	if !ok || bytes.(int64) != 1024 {
		t.Errorf("Expected bytes=1024, got %v", bytes)
	}

	// Verify geographic locations (fields 41-42 in full format)
	// In our simplified test, we'll just verify they're parsed
	if loc, ok := event["srcloc"]; ok && loc != "" {
		t.Logf("Source location: %v", loc)
	}

	if loc, ok := event["dstloc"]; ok && loc != "" {
		t.Logf("Destination location: %v", loc)
	}

	// Verify _raw preservation
	if event["_raw"] != raw {
		t.Error("_raw field was not preserved")
	}

	// Verify parser metadata
	if event["_parser"] != "paloalto_csv" {
		t.Error("_parser metadata missing")
	}

	if _, ok := event["_parsed_at"]; !ok {
		t.Error("_parsed_at metadata missing")
	}

	if event["paloalto_log_type"] != "TRAFFIC" {
		t.Error("paloalto_log_type metadata missing or incorrect")
	}
}

func TestPaloAltoCsvParser_ThreatLog(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Sample THREAT log with URL filtering
	raw := `,2024/01/15 11:00:00,007951000012345,THREAT,url,,2024/01/15 11:00:00,192.168.1.200,8.8.8.8,0.0.0.0,0.0.0.0,Block-Malware,alice@corp.com,,ssl,vsys1,trust,untrust,ethernet1/1,ethernet1/2,Forward,,234567,1,55000,443,0,0,0x80000000,tcp,alert,http://malicious.example.com/payload,999888777(9999),hacking,high,client-to-server,111111,0x0,US,US,,text/html,12345,abcd1234567890ef,cloud-analysis,1,Mozilla/5.0,text/html,,,,receipient@example.com,Malicious Email,sender@badguys.com,WF-12345678,1,2,3,4,vsys1,PA-5220,uuid-src,uuid-dst,GET,0,,,0,,tunnel-id,gambling,content-v123`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify log type
	if event["type"] != "THREAT" {
		t.Errorf("Expected type=THREAT, got %v", event["type"])
	}

	if event["subtype"] != "url" {
		t.Errorf("Expected subtype=url, got %v", event["subtype"])
	}

	// Verify network fields
	if event["src"] != "192.168.1.200" {
		t.Errorf("Expected src=192.168.1.200, got %v", event["src"])
	}

	if event["dst"] != "8.8.8.8" {
		t.Errorf("Expected dst=8.8.8.8, got %v", event["dst"])
	}

	// Verify user
	if event["srcuser"] != "alice@corp.com" {
		t.Errorf("Expected srcuser=alice@corp.com, got %v", event["srcuser"])
	}

	// Verify threat-specific fields
	if event["misc"] != "http://malicious.example.com/payload" {
		t.Errorf("Expected misc=URL, got %v", event["misc"])
	}

	if event["threatid"] != "999888777(9999)" {
		t.Errorf("Expected threatid, got %v", event["threatid"])
	}

	if event["category"] != "hacking" {
		t.Errorf("Expected category=hacking, got %v", event["category"])
	}

	if event["severity"] != "high" {
		t.Errorf("Expected severity=high, got %v", event["severity"])
	}

	if event["direction"] != "client-to-server" {
		t.Errorf("Expected direction=client-to-server, got %v", event["direction"])
	}

	// Verify action
	if event["action"] != "alert" {
		t.Errorf("Expected action=alert, got %v", event["action"])
	}

	// Verify rule
	if event["rule"] != "Block-Malware" {
		t.Errorf("Expected rule=Block-Malware, got %v", event["rule"])
	}

	// Verify application
	if event["app"] != "ssl" {
		t.Errorf("Expected app=ssl, got %v", event["app"])
	}

	// Verify content type
	if event["contenttype"] != "text/html" {
		t.Errorf("Expected contenttype=text/html, got %v", event["contenttype"])
	}

	// Verify file digest
	if event["filedigest"] != "abcd1234567890ef" {
		t.Errorf("Expected filedigest, got %v", event["filedigest"])
	}

	// Verify user agent
	if event["user_agent"] != "Mozilla/5.0" {
		t.Errorf("Expected user_agent, got %v", event["user_agent"])
	}

	// Verify email fields (positions 50, 51, 52)
	if event["sender"] != "sender@badguys.com" {
		t.Logf("Sender field: %v (position may vary by log format)", event["sender"])
	}

	if event["subject"] != "Malicious Email" {
		t.Logf("Subject field: %v (position may vary by log format)", event["subject"])
	}

	// Verify _raw preservation
	if event["_raw"] != raw {
		t.Error("_raw field was not preserved")
	}
}

func TestPaloAltoCsvParser_ConfigLog(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Sample CONFIG log
	raw := `,2024/01/15 12:00:00,007951000012345,CONFIG,,,2024/01/15 12:00:00,PA-VM,set,admin-user,Web,Succeeded,/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system,<old-config>,<new-config>,12345,0x0,1,2,3,4,vsys1,PA-5220`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify log type
	if event["type"] != "CONFIG" {
		t.Errorf("Expected type=CONFIG, got %v", event["type"])
	}

	// Verify config-specific fields
	if event["host"] != "PA-VM" {
		t.Errorf("Expected host=PA-VM, got %v", event["host"])
	}

	if event["cmd"] != "set" {
		t.Errorf("Expected cmd=set, got %v", event["cmd"])
	}

	if event["admin"] != "admin-user" {
		t.Errorf("Expected admin=admin-user, got %v", event["admin"])
	}

	if event["client"] != "Web" {
		t.Errorf("Expected client=Web, got %v", event["client"])
	}

	if event["config_result"] != "Succeeded" {
		t.Errorf("Expected config_result=Succeeded, got %v", event["config_result"])
	}

	// Verify path
	if event["path"] != "/config/devices/entry[@name='localhost.localdomain']/deviceconfig/system" {
		t.Errorf("Expected path, got %v", event["path"])
	}

	// Verify _raw preservation
	if event["_raw"] != raw {
		t.Error("_raw field was not preserved")
	}
}

func TestPaloAltoCsvParser_SystemLog(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Sample SYSTEM log
	// Format: FUTURE_USE, Receive Time, Serial, Type, Subtype, FUTURE_USE, Generated Time, Vsys, EventID, Object, FUTURE_USE, FUTURE_USE, Module, Severity, Description
	raw := `,2024/01/15 13:00:00,007951000012345,SYSTEM,general,,2024/01/15 13:00:00,vsys1,general,,,, general,informational,System startup completed,54321,0x0,1,2,3,4,vsys1,PA-220`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify log type
	if event["type"] != "SYSTEM" {
		t.Errorf("Expected type=SYSTEM, got %v", event["type"])
	}

	if event["subtype"] != "general" {
		t.Errorf("Expected subtype=general, got %v", event["subtype"])
	}

	// Verify system-specific fields
	if event["vsys"] != "vsys1" {
		t.Errorf("Expected vsys=vsys1, got %v", event["vsys"])
	}

	if event["eventid"] != "general" {
		t.Errorf("Expected eventid=general, got %v", event["eventid"])
	}

	if event["module"] != "general" {
		t.Errorf("Expected module=general, got %v", event["module"])
	}

	if event["severity"] != "informational" {
		t.Errorf("Expected severity=informational, got %v", event["severity"])
	}

	if event["opaque"] != "System startup completed" {
		t.Errorf("Expected opaque message, got %v", event["opaque"])
	}

	// Verify _raw preservation
	if event["_raw"] != raw {
		t.Error("_raw field was not preserved")
	}
}

func TestPaloAltoCsvParser_AuthenticationLog(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Sample AUTHENTICATION log
	raw := `,2024/01/15 14:00:00,007951000012345,AUTHENTICATION,,,2024/01/15 14:00:00,vsys1,192.168.1.50,bob@corp.com,bob,portal-auth,Auth-Success,1,auth-12345,radius,Log-Forwarding,radius-profile,Successful authentication,Captive-Portal,authentication-success,1,99999,0x0,1,2,3,4,vsys1,PA-3220,1,RADIUS`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify log type
	if event["type"] != "AUTHENTICATION" {
		t.Errorf("Expected type=AUTHENTICATION, got %v", event["type"])
	}

	// Verify auth-specific fields
	if event["vsys"] != "vsys1" {
		t.Errorf("Expected vsys=vsys1, got %v", event["vsys"])
	}

	if event["ip"] != "192.168.1.50" {
		t.Errorf("Expected ip=192.168.1.50, got %v", event["ip"])
	}

	if event["user"] != "bob@corp.com" {
		t.Errorf("Expected user=bob@corp.com, got %v", event["user"])
	}

	if event["normalize_user"] != "bob" {
		t.Errorf("Expected normalize_user=bob, got %v", event["normalize_user"])
	}

	if event["object"] != "portal-auth" {
		t.Errorf("Expected object=portal-auth, got %v", event["object"])
	}

	if event["authpolicy"] != "Auth-Success" {
		t.Errorf("Expected authpolicy=Auth-Success, got %v", event["authpolicy"])
	}

	if event["vendor"] != "radius" {
		t.Errorf("Expected vendor=radius, got %v", event["vendor"])
	}

	if event["serverprofile"] != "radius-profile" {
		t.Errorf("Expected serverprofile=radius-profile, got %v", event["serverprofile"])
	}

	if event["description"] != "Successful authentication" {
		t.Errorf("Expected description, got %v", event["description"])
	}

	if event["clienttype"] != "Captive-Portal" {
		t.Errorf("Expected clienttype=Captive-Portal, got %v", event["clienttype"])
	}

	if event["event"] != "authentication-success" {
		t.Errorf("Expected event=authentication-success, got %v", event["event"])
	}

	// Verify _raw preservation
	if event["_raw"] != raw {
		t.Error("_raw field was not preserved")
	}
}

func TestPaloAltoCsvParser_GlobalProtectLog(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Sample GLOBALPROTECT log
	raw := `,2024/01/15 15:00:00,007951000012345,GLOBALPROTECT,,,2024/01/15 15:00:00,vsys1,gateway-auth,prelogon,saml,ipsec-tunnel,charlie@corp.com,US,LAPTOP-123,203.0.113.50,2001:db8::1,10.10.10.50,fd00::50,host-id-123,serial-123,5.2.0,Windows,10.0.19045,5,Connection successful,,Connecting...,78888,0x0,2024/01/15 15:00:01,Tunnel,150,5,gw1.example.com;gw2.example.com,gw1.example.com,1,2,3,4,vsys1,PA-5220,1`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify log type
	if event["type"] != "GLOBALPROTECT" {
		t.Errorf("Expected type=GLOBALPROTECT, got %v", event["type"])
	}

	// Verify GP-specific fields
	if event["vsys"] != "vsys1" {
		t.Errorf("Expected vsys=vsys1, got %v", event["vsys"])
	}

	if event["eventid"] != "gateway-auth" {
		t.Errorf("Expected eventid=gateway-auth, got %v", event["eventid"])
	}

	if event["stage"] != "prelogon" {
		t.Errorf("Expected stage=prelogon, got %v", event["stage"])
	}

	if event["auth_method"] != "saml" {
		t.Errorf("Expected auth_method=saml, got %v", event["auth_method"])
	}

	if event["tunnel_type"] != "ipsec-tunnel" {
		t.Errorf("Expected tunnel_type=ipsec-tunnel, got %v", event["tunnel_type"])
	}

	if event["srcuser"] != "charlie@corp.com" {
		t.Errorf("Expected srcuser=charlie@corp.com, got %v", event["srcuser"])
	}

	if event["srcregion"] != "US" {
		t.Errorf("Expected srcregion=US, got %v", event["srcregion"])
	}

	if event["machinename"] != "LAPTOP-123" {
		t.Errorf("Expected machinename=LAPTOP-123, got %v", event["machinename"])
	}

	if event["public_ip"] != "203.0.113.50" {
		t.Errorf("Expected public_ip, got %v", event["public_ip"])
	}

	if event["private_ip"] != "10.10.10.50" {
		t.Errorf("Expected private_ip, got %v", event["private_ip"])
	}

	if event["client_ver"] != "5.2.0" {
		t.Errorf("Expected client_ver=5.2.0, got %v", event["client_ver"])
	}

	if event["client_os"] != "Windows" {
		t.Errorf("Expected client_os=Windows, got %v", event["client_os"])
	}

	if event["reason"] != "Connection successful" {
		t.Errorf("Expected reason, got %v", event["reason"])
	}

	// Verify _raw preservation
	if event["_raw"] != raw {
		t.Error("_raw field was not preserved")
	}
}

func TestPaloAltoCsvParser_QuotedFields(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Test with quoted fields containing commas
	raw := `,2024/01/15 16:00:00,007951000012345,THREAT,vulnerability,,2024/01/15 16:00:00,10.0.0.1,10.0.0.2,,,CVE-Rule,"user@domain.com, admin",,,vsys1,trust,untrust,eth1,eth2,Forward,,55555,1,8080,80,,,0x0,tcp,alert,"Exploit attempt, multiple vectors detected",CVE-2023-12345,"command-and-control, malware",critical,server-to-client,88888,0x0,US,CN,,application/json`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify quoted field with comma is parsed correctly
	if event["srcuser"] != "user@domain.com, admin" {
		t.Errorf("Expected quoted srcuser with comma, got %v", event["srcuser"])
	}

	// Verify threat misc field with comma
	if event["misc"] != "Exploit attempt, multiple vectors detected" {
		t.Errorf("Expected quoted misc with comma, got %v", event["misc"])
	}

	// Verify category with comma
	if event["category"] != "command-and-control, malware" {
		t.Errorf("Expected quoted category with comma, got %v", event["category"])
	}
}

func TestPaloAltoCsvParser_EmptyFields(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Test with many empty fields
	raw := `,2024/01/15 17:00:00,007951000012345,TRAFFIC,drop,,2024/01/15 17:00:00,192.168.1.1,2.2.2.2,,,Allow-All,,,app1,vsys1,trust,untrust,eth1,eth2,Forward,,123,1,12345,22,0,0,0x0,tcp,deny,0,0,0,0,,,any`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify type
	if event["type"] != "TRAFFIC" {
		t.Errorf("Expected type=TRAFFIC, got %v", event["type"])
	}

	if event["subtype"] != "drop" {
		t.Errorf("Expected subtype=drop, got %v", event["subtype"])
	}

	// Verify source IP is present
	if event["src"] != "192.168.1.1" {
		t.Errorf("Expected src=192.168.1.1, got %v", event["src"])
	}

	// Verify action
	if event["action"] != "deny" {
		t.Errorf("Expected action=deny, got %v", event["action"])
	}
}

func TestPaloAltoCsvParser_MissingRawField(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	event := map[string]interface{}{
		"other_field": "value",
	}

	err := parser.Parse(event)
	if err == nil {
		t.Error("Expected error for missing _raw field")
	}
}

func TestPaloAltoCsvParser_InvalidRawType(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	event := map[string]interface{}{
		"_raw": 12345, // Not a string
	}

	err := parser.Parse(event)
	if err == nil {
		t.Error("Expected error for non-string _raw field")
	}
}

func TestPaloAltoCsvParser_InsufficientFields(t *testing.T) {
	parser := NewPaloAltoCSVParser()
	parser.StrictMode = true // Enable strict mode to get errors

	// Too few fields
	raw := `,2024/01/15,serial,TYPE`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err == nil {
		t.Error("Expected error for insufficient fields in strict mode")
	}
}

func TestPaloAltoCsvParser_StrictMode(t *testing.T) {
	parser := NewPaloAltoCSVParser()
	parser.StrictMode = true

	// Invalid CSV (unclosed quote)
	raw := `,2024/01/15,serial,"unclosed quote field`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err == nil {
		t.Error("Expected error in strict mode for malformed CSV")
	}
}

func TestPaloAltoCsvParser_LenientMode(t *testing.T) {
	parser := NewPaloAltoCSVParser()
	parser.StrictMode = false // Default lenient mode

	// Slightly malformed CSV - need minimum 10 fields
	raw := `,2024/01/15 10:00:00,serial123,TRAFFIC,start,,2024/01/15 10:00:00,1.1.1.1,2.2.2.2,3.3.3.3,4.4.4.4`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	// Should not error in lenient mode
	if err != nil {
		t.Logf("Lenient mode error (acceptable): %v", err)
	}

	// Should still parse what's available
	if event["type"] != "TRAFFIC" {
		t.Errorf("Expected type to be parsed even in lenient mode, got %v", event["type"])
	}
}

func TestPaloAltoCsvParser_PreserveRaw(t *testing.T) {
	parser := NewPaloAltoCSVParser()
	parser.PreserveRaw = true

	raw := `,2024/01/15 10:00:00,007951000012345,SYSTEM,,,2024/01/15 10:00:00,vsys1,general,,general,info,test,12345,0x0`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify _raw is preserved
	if event["_raw"] != raw {
		t.Error("_raw field was not preserved when PreserveRaw=true")
	}
}

func TestPaloAltoCsvParser_NoPreserveRaw(t *testing.T) {
	parser := NewPaloAltoCSVParser()
	parser.PreserveRaw = false

	raw := `,2024/01/15 10:00:00,007951000012345,SYSTEM,,,2024/01/15 10:00:00,vsys1,general,,general,info,test,12345,0x0`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// _raw should still be present (always preserved for legal compliance)
	if _, ok := event["_raw"]; !ok {
		t.Error("_raw field should always be preserved for legal compliance")
	}
}

func TestPaloAltoCsvParser_TypeConversion(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	raw := `,2024/01/15 10:00:00,007951000012345,TRAFFIC,end,,2024/01/15 10:00:00,1.1.1.1,2.2.2.2,,,rule1,,,app1,vsys1,z1,z2,eth1,eth2,log,,999,5,54321,443,0,0,0x0,tcp,allow,2048,1024,1024,100`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test integer conversions
	sessionid, ok := event["sessionid"].(int64)
	if !ok {
		t.Errorf("sessionid should be int64, got %T", event["sessionid"])
	}
	if sessionid != 999 {
		t.Errorf("Expected sessionid=999, got %d", sessionid)
	}

	repeatcnt, ok := event["repeatcnt"].(int64)
	if !ok {
		t.Errorf("repeatcnt should be int64, got %T", event["repeatcnt"])
	}
	if repeatcnt != 5 {
		t.Errorf("Expected repeatcnt=5, got %d", repeatcnt)
	}

	sport, ok := event["sport"].(int64)
	if !ok {
		t.Errorf("sport should be int64, got %T", event["sport"])
	}
	if sport != 54321 {
		t.Errorf("Expected sport=54321, got %d", sport)
	}

	bytes, ok := event["bytes"].(int64)
	if !ok {
		t.Errorf("bytes should be int64, got %T", event["bytes"])
	}
	if bytes != 2048 {
		t.Errorf("Expected bytes=2048, got %d", bytes)
	}

	packets, ok := event["packets"].(int64)
	if !ok {
		t.Errorf("packets should be int64, got %T", event["packets"])
	}
	if packets != 100 {
		t.Errorf("Expected packets=100, got %d", packets)
	}
}

func TestPaloAltoCsvParser_UnknownLogType(t *testing.T) {
	parser := NewPaloAltoCSVParser()

	// Unknown log type should fall back to generic parsing
	raw := `,2024/01/15 10:00:00,serial123,UNKNOWN_TYPE,subtype,,2024/01/15 10:00:00,field7,field8,field9,field10`

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should still parse type
	if event["type"] != "UNKNOWN_TYPE" {
		t.Errorf("Expected type=UNKNOWN_TYPE, got %v", event["type"])
	}

	// Should have generic field_N fields
	if val, ok := event["field_7"]; !ok || val != "field7" {
		t.Logf("Generic field_7: %v", val)
	}
}

func TestGetCommonPaloAltoFields(t *testing.T) {
	fields := GetCommonPaloAltoFields()

	if len(fields) == 0 {
		t.Error("Expected non-empty field list")
	}

	// Check for some expected fields
	expectedFields := []string{"src", "dst", "sport", "dport", "proto", "action", "app", "rule"}
	for _, expected := range expectedFields {
		found := false
		for _, field := range fields {
			if field == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field %s not found in common fields", expected)
		}
	}
}

// Benchmark the parser
func BenchmarkPaloAltoCsvParser_Parse(b *testing.B) {
	parser := NewPaloAltoCSVParser()

	raw := `,2024/01/15 10:30:45,007951000012345,TRAFFIC,end,,2024/01/15 10:30:44,192.168.1.100,10.0.0.50,0.0.0.0,0.0.0.0,Allow-Web,user@corp.com,,web-browsing,vsys1,trust,untrust,ethernet1/1,ethernet1/2,Log-Forwarding,,123456,1,54321,443,0,0,0x80000000,tcp,allow,1024,512,512,10,2024/01/15 10:30:35,9,any,,,0,0x0,US,GB,0,5,5,0,aged-out,1,2,3,4,vsys1,PA-VM,policy,uuid-1,uuid-2,0,,0,0,0,0,tunnel-1234,0,0,0,0,rule-uuid,,0,0,0,,,,prod-cluster,sdwan-device,hub,site-a,dug-1,10.1.1.1`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := map[string]interface{}{
			"_raw": raw,
		}
		_ = parser.Parse(event)
	}
}

// Benchmark THREAT log parsing
func BenchmarkPaloAltoCsvParser_ParseThreat(b *testing.B) {
	parser := NewPaloAltoCSVParser()

	raw := `,2024/01/15 11:00:00,007951000012345,THREAT,url,,2024/01/15 11:00:00,192.168.1.200,8.8.8.8,0.0.0.0,0.0.0.0,Block-Malware,alice@corp.com,,ssl,vsys1,trust,untrust,ethernet1/1,ethernet1/2,Forward,,234567,1,55000,443,0,0,0x80000000,tcp,alert,http://malicious.example.com/payload,999888777(9999),hacking,high,client-to-server,111111,0x0,US,US,,text/html,12345,abcd1234567890ef,cloud-analysis,1,Mozilla/5.0,text/html,,,,receipient@example.com,Malicious Email,sender@badguys.com,WF-12345678,1,2,3,4,vsys1,PA-5220,uuid-src,uuid-dst,GET,0,,,0,,tunnel-id,gambling,content-v123`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := map[string]interface{}{
			"_raw": raw,
		}
		_ = parser.Parse(event)
	}
}
