package filters

import (
	"testing"
)

func TestVersaKVPParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "flowIdLog with basic fields",
			raw:  "2017-11-26T22:42:38+0000 flowIdLog, applianceName=Branch1, tenantName=Customer1, flowId=33655871, flowCookie=1511734794, sourceIPv4Address=172.21.1.2, destinationIPv4Address=172.21.2.2, sourcePort=44657, destinationPort=5001",
			want: map[string]interface{}{
				"_log_type":              "flowIdLog",
				"applianceName":          "Branch1",
				"tenantName":             "Customer1",
				"flowId":                 int64(33655871),
				"flowCookie":             int64(1511734794),
				"sourceIPv4Address":      "172.21.1.2",
				"destinationIPv4Address": "172.21.2.2",
				"sourcePort":             int64(44657),
				"destinationPort":        int64(5001),
			},
			wantErr: false,
		},
		{
			name: "accessLog with quoted values",
			raw:  `2021-03-18T16:00:17+0000 accessLog, applianceName=SDWAN-Branch4, tenantName=Tenant1, flowId=2181092523, action=allow, rule=Allow_From_Trust, host=www.netflix.com, appIdStr=netflix, fromUser=user123@versa-networks.com`,
			want: map[string]interface{}{
				"_log_type":     "accessLog",
				"applianceName": "SDWAN-Branch4",
				"tenantName":    "Tenant1",
				"flowId":        int64(2181092523),
				"action":        "allow",
				"rule":          "Allow_From_Trust",
				"host":          "www.netflix.com",
				"appIdStr":      "netflix",
				"fromUser":      "user123@versa-networks.com",
			},
			wantErr: false,
		},
		{
			name: "idpLog with signature message",
			raw:  `2024-07-11T02:11:06+0000 idpLog, applianceName=Branch1, tenantName=USA, flowId=41532610, flowCookie=1720663865, signatureId=1061212062, idpAction=alert, signatureMsg="Microsoft Windows SNMP Service Memory Corruption", classMsg="Attempted User Privilege Gain", threatType=attempted-user`,
			want: map[string]interface{}{
				"_log_type":     "idpLog",
				"applianceName": "Branch1",
				"tenantName":    "USA",
				"flowId":        int64(41532610),
				"flowCookie":    int64(1720663865),
				"signatureId":   int64(1061212062),
				"idpAction":     "alert",
				"signatureMsg":  "Microsoft Windows SNMP Service Memory Corruption",
				"classMsg":      "Attempted User Privilege Gain",
				"threatType":    "attempted-user",
			},
			wantErr: false,
		},
		{
			name: "urlfLog with URL",
			raw:  `2021-02-18T18:50:15+0000 urlfLog, applianceName=SDWAN-Branch1, tenantName=Tenant1, flowId=3254966030, flowCookie=1613674373, urlReputation=trustworthy, urlCategory=streaming_media, httpUrl=www.youtube.com/index.html, urlfProfile=YoutubeRule, urlfAction=alert`,
			want: map[string]interface{}{
				"_log_type":     "urlfLog",
				"applianceName": "SDWAN-Branch1",
				"tenantName":    "Tenant1",
				"flowId":        int64(3254966030),
				"flowCookie":    int64(1613674373),
				"urlReputation": "trustworthy",
				"urlCategory":   "streaming_media",
				"httpUrl":       "www.youtube.com/index.html",
				"urlfProfile":   "YoutubeRule",
				"urlfAction":    "alert",
			},
			wantErr: false,
		},
		{
			name: "avLog with malware detection",
			raw:  `2021-02-20T07:45:41+0000 avLog, applianceName=SDWAN-Branch1, tenantName=Tenant3, flowId=12345, fileName=eicarcom2.zip, fileType=zip, fileTransDir=download, avMalwareName=EICAR_Test_File, avAction=deny, threatType=virus_event, threatSeverity=critical`,
			want: map[string]interface{}{
				"_log_type":      "avLog",
				"applianceName":  "SDWAN-Branch1",
				"tenantName":     "Tenant3",
				"flowId":         int64(12345),
				"fileName":       "eicarcom2.zip",
				"fileType":       "zip",
				"fileTransDir":   "download",
				"avMalwareName":  "EICAR_Test_File",
				"avAction":       "deny",
				"threatType":     "virus_event",
				"threatSeverity": "critical",
			},
			wantErr: false,
		},
		{
			name: "cgnatLog with NAT translation",
			raw:  `2017-11-26T22:36:31+0000 cgnatLog, applianceName=DC1Branch1, tenantName=Customer1, flowId=33889107, sourceIPv4Address=172.18.101.10, destinationIPv4Address=8.8.8.8, postNATSourceIPv4Address=70.70.5.2, sourcePort=37190, destinationPort=53, natRuleName=DIA-Rule-Customer1-LAN1-VR-ISPA-Network, natEvent=nat44-sess-create`,
			want: map[string]interface{}{
				"_log_type":                "cgnatLog",
				"applianceName":            "DC1Branch1",
				"tenantName":               "Customer1",
				"flowId":                   int64(33889107),
				"sourceIPv4Address":        "172.18.101.10",
				"destinationIPv4Address":   "8.8.8.8",
				"postNATSourceIPv4Address": "70.70.5.2",
				"sourcePort":               int64(37190),
				"destinationPort":          int64(53),
				"natRuleName":              "DIA-Rule-Customer1-LAN1-VR-ISPA-Network",
				"natEvent":                 "nat44-sess-create",
			},
			wantErr: false,
		},
		{
			name: "dnsfLog with DNS filtering",
			raw:  `2024-01-23T14:05:48+0000 dnsfLog, applianceName=SDWAN-Branch1, tenantName=Tenant1, flowId=33554458, dnsfProfileName=dnsfilter_profile, dnsfMsgType=request, dnsfEvType=blacklist, dnsfAction=drop-packet, dnsfDomain="www.facebook.com", sourceIPv4Address=172.16.11.10, destinationIPv4Address=10.48.0.99`,
			want: map[string]interface{}{
				"_log_type":              "dnsfLog",
				"applianceName":          "SDWAN-Branch1",
				"tenantName":             "Tenant1",
				"flowId":                 int64(33554458),
				"dnsfProfileName":        "dnsfilter_profile",
				"dnsfMsgType":            "request",
				"dnsfEvType":             "blacklist",
				"dnsfAction":             "drop-packet",
				"dnsfDomain":             "www.facebook.com",
				"sourceIPv4Address":      "172.16.11.10",
				"destinationIPv4Address": "10.48.0.99",
			},
			wantErr: false,
		},
		{
			name: "empty values",
			raw:  `2024-01-01T00:00:00+0000 testLog, applianceName=Test, emptyField=, normalField=value`,
			want: map[string]interface{}{
				"_log_type":     "testLog",
				"applianceName": "Test",
				"emptyField":    "",
				"normalField":   "value",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewVersaKVPParser()
			event := map[string]interface{}{
				"_raw": tt.raw,
			}

			err := parser.Parse(event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check that _raw is preserved
			if event["_raw"] != tt.raw {
				t.Errorf("_raw field not preserved: got %v, want %v", event["_raw"], tt.raw)
			}

			// Check expected fields
			for key, wantVal := range tt.want {
				gotVal, ok := event[key]
				if !ok {
					t.Errorf("missing field %q", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("field %q = %v (%T), want %v (%T)", key, gotVal, gotVal, wantVal, wantVal)
				}
			}

			// Check metadata fields
			if _, ok := event["_parser"]; !ok {
				t.Error("missing _parser metadata field")
			}
			if _, ok := event["@timestamp"]; !ok {
				t.Error("missing @timestamp field")
			}
		})
	}
}

func TestVersaKVPParser_SplitKVPs(t *testing.T) {
	parser := NewVersaKVPParser()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple KVPs",
			input: "key1=value1, key2=value2, key3=value3",
			want:  []string{"key1=value1", "key2=value2", "key3=value3"},
		},
		{
			name:  "quoted value with comma",
			input: `key1="value, with, commas", key2=simple`,
			want:  []string{`key1="value, with, commas"`, "key2=simple"},
		},
		{
			name:  "quoted value with equals",
			input: `key1="value=with=equals", key2=normal`,
			want:  []string{`key1="value=with=equals"`, "key2=normal"},
		},
		{
			name:  "empty values",
			input: "key1=, key2=value, key3=",
			want:  []string{"key1=", "key2=value", "key3="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.splitKVPs(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitKVPs() got %d items, want %d items", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitKVPs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestVersaKVPParser_ParseKeyValue(t *testing.T) {
	parser := NewVersaKVPParser()

	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
	}{
		{
			name:      "simple KVP",
			input:     "key=value",
			wantKey:   "key",
			wantValue: "value",
		},
		{
			name:      "quoted value",
			input:     `key="quoted value"`,
			wantKey:   "key",
			wantValue: "quoted value",
		},
		{
			name:      "value with equals",
			input:     `url="http://example.com?a=1&b=2"`,
			wantKey:   "url",
			wantValue: "http://example.com?a=1&b=2",
		},
		{
			name:      "escaped quotes",
			input:     `msg="He said \"hello\""`,
			wantKey:   "msg",
			wantValue: `He said "hello"`,
		},
		{
			name:      "empty value",
			input:     "key=",
			wantKey:   "key",
			wantValue: "",
		},
		{
			name:      "no equals",
			input:     "invalid",
			wantKey:   "",
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotValue := parser.parseKeyValue(tt.input)
			if gotKey != tt.wantKey {
				t.Errorf("parseKeyValue() key = %q, want %q", gotKey, tt.wantKey)
			}
			if gotValue != tt.wantValue {
				t.Errorf("parseKeyValue() value = %q, want %q", gotValue, tt.wantValue)
			}
		})
	}
}

func TestVersaKVPParser_PreserveRaw(t *testing.T) {
	raw := "2017-11-26T22:42:38+0000 flowIdLog, applianceName=Branch1, tenantName=Customer1"

	parser := NewVersaKVPParser()
	parser.PreserveRaw = true

	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify _raw is preserved
	if event["_raw"] != raw {
		t.Errorf("_raw not preserved when PreserveRaw=true")
	}

	// Now test with PreserveRaw = false
	parser.PreserveRaw = false
	event2 := map[string]interface{}{
		"_raw": raw,
	}

	err = parser.Parse(event2)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// _raw should still be there (we always preserve for legal compliance)
	if _, ok := event2["_raw"]; !ok {
		t.Error("_raw removed even though it's required for legal compliance")
	}
}

func TestVersaKVPParser_TypeConversion(t *testing.T) {
	raw := "2024-01-01T00:00:00+0000 testLog, flowId=123456, sourcePort=443, latency=12.5, name=test"

	parser := NewVersaKVPParser()
	event := map[string]interface{}{
		"_raw": raw,
	}

	err := parser.Parse(event)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check integer conversion
	if flowID, ok := event["flowId"].(int64); !ok {
		t.Errorf("flowId not converted to int64, got type %T", event["flowId"])
	} else if flowID != 123456 {
		t.Errorf("flowId = %d, want 123456", flowID)
	}

	// Check port conversion
	if port, ok := event["sourcePort"].(int64); !ok {
		t.Errorf("sourcePort not converted to int64, got type %T", event["sourcePort"])
	} else if port != 443 {
		t.Errorf("sourcePort = %d, want 443", port)
	}

	// Check string remains string
	if name, ok := event["name"].(string); !ok {
		t.Errorf("name not string, got type %T", event["name"])
	} else if name != "test" {
		t.Errorf("name = %q, want %q", name, "test")
	}
}

func BenchmarkVersaKVPParser_Parse(b *testing.B) {
	parser := NewVersaKVPParser()
	raw := `2024-07-11T02:11:06+0000 idpLog, applianceName=Branch1, tenantName=USA, flowId=41532610, flowCookie=1720663865, signatureId=1061212062, groupId=1, signatureRev=0, vsnId=0, applianceId=1, tenantId=2, moduleId=10, signaturePriority=critical, idpAction=alert, signatureMsg="Microsoft Windows SNMP Service Memory Corruption", classMsg="Attempted User Privilege Gain", threatType=attempted-user, sourceIPv4Address=10.205.167.170, destinationIPv4Address=10.191.64.21, sourceTransportPort=44924, destinationTransportPort=161, protocolIdentifier=17`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := map[string]interface{}{
			"_raw": raw,
		}
		_ = parser.Parse(event)
	}
}
