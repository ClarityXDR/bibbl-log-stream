package filters

import (
	"testing"
)

func TestUniversalKVParser_Parse(t *testing.T) {
	parser := NewUniversalKVParser()

	tests := []struct {
		name     string
		input    map[string]interface{}
		wantKeys []string
		wantVals map[string]string
	}{
		{
			name: "simple key=value pairs",
			input: map[string]interface{}{
				"message": "src=10.0.0.1 dst=192.168.1.1 action=allow proto=tcp",
			},
			wantKeys: []string{"src", "dst", "action", "proto"},
			wantVals: map[string]string{
				"src":    "10.0.0.1",
				"dst":    "192.168.1.1",
				"action": "allow",
				"proto":  "tcp",
			},
		},
		{
			name: "quoted values",
			input: map[string]interface{}{
				"message": `user="john doe" action="login attempt" result=success`,
			},
			wantKeys: []string{"user", "action", "result"},
			wantVals: map[string]string{
				"user":   "john doe",
				"action": "login attempt",
				"result": "success",
			},
		},
		{
			name: "mixed delimiters",
			input: map[string]interface{}{
				"message": "src=10.0.0.1,dst=10.0.0.2;port=443|proto=https",
			},
			wantKeys: []string{"src", "dst", "port", "proto"},
			wantVals: map[string]string{
				"src":   "10.0.0.1",
				"dst":   "10.0.0.2",
				"port":  "443",
				"proto": "https",
			},
		},
		{
			name: "keys with dots and dashes",
			input: map[string]interface{}{
				"message": "src.ip=10.0.0.1 dst-port=443 user_name=admin",
			},
			wantKeys: []string{"src_ip", "dst_port", "user_name"},
			wantVals: map[string]string{
				"src_ip":    "10.0.0.1",
				"dst_port":  "443",
				"user_name": "admin",
			},
		},
		{
			name: "severity normalization - critical",
			input: map[string]interface{}{
				"message": "severity=critical src=10.0.0.1 action=block",
			},
			wantKeys: []string{"src", "action", "severity"},
			wantVals: map[string]string{
				"severity": "critical",
			},
		},
		{
			name: "severity normalization - high from error",
			input: map[string]interface{}{
				"message": "level=error src=10.0.0.1 action=deny",
			},
			wantKeys: []string{"src", "action", "severity"},
			wantVals: map[string]string{
				"severity": "high",
			},
		},
		{
			name: "severity normalization - medium from warning",
			input: map[string]interface{}{
				"message": "priority=warning src=10.0.0.1",
			},
			wantKeys: []string{"src", "severity"},
			wantVals: map[string]string{
				"severity": "medium",
			},
		},
		{
			name: "single quoted values",
			input: map[string]interface{}{
				"message": "user='john doe' msg='test message'",
			},
			wantKeys: []string{"user", "msg"},
			wantVals: map[string]string{
				"user": "john doe",
				"msg":  "test message",
			},
		},
		{
			name: "empty message",
			input: map[string]interface{}{
				"message": "",
			},
			wantKeys: []string{},
			wantVals: map[string]string{},
		},
		{
			name: "raw field instead of message",
			input: map[string]interface{}{
				"raw": "src=1.2.3.4 dst=5.6.7.8",
			},
			wantKeys: []string{"src", "dst"},
			wantVals: map[string]string{
				"src": "1.2.3.4",
				"dst": "5.6.7.8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.Parse(tt.input)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			// Check expected keys exist
			for _, key := range tt.wantKeys {
				if _, ok := tt.input[key]; !ok {
					t.Errorf("Expected key %q not found in result", key)
				}
			}

			// Check expected values
			for key, wantVal := range tt.wantVals {
				gotVal, ok := tt.input[key]
				if !ok {
					t.Errorf("Key %q not found", key)
					continue
				}
				if got, ok := gotVal.(string); ok {
					if got != wantVal {
						t.Errorf("Key %q = %q, want %q", key, got, wantVal)
					}
				}
			}
		})
	}
}

func TestUniversalKVParser_RealWorldFormats(t *testing.T) {
	parser := NewUniversalKVParser()

	tests := []struct {
		name    string
		message string
		check   func(t *testing.T, event map[string]interface{})
	}{
		{
			name:    "Palo Alto style",
			message: `TRAFFIC,2024/01/15 10:30:45,src=192.168.1.100,dst=10.0.0.1,srcport=54321,dstport=443,protocol=tcp,action=allow,rule="allow-web"`,
			check: func(t *testing.T, event map[string]interface{}) {
				if event["src"] != "192.168.1.100" {
					t.Errorf("src = %v, want 192.168.1.100", event["src"])
				}
				if event["action"] != "allow" {
					t.Errorf("action = %v, want allow", event["action"])
				}
			},
		},
		{
			name:    "FortiGate style",
			message: `date=2024-01-15 time=10:30:45 devname="FGT-01" devid="FG500E" srcip=192.168.1.50 dstip=8.8.8.8 service="HTTPS" action=accept`,
			check: func(t *testing.T, event map[string]interface{}) {
				if event["devname"] != "FGT-01" {
					t.Errorf("devname = %v, want FGT-01", event["devname"])
				}
				if event["srcip"] != "192.168.1.50" {
					t.Errorf("srcip = %v, want 192.168.1.50", event["srcip"])
				}
			},
		},
		{
			name:    "Cisco ASA style",
			message: `%ASA-6-302013: Built outbound TCP connection 12345 for outside:192.168.1.1/443 src=10.0.0.50/12345 dst=192.168.1.1/443`,
			check: func(t *testing.T, event map[string]interface{}) {
				if event["src"] != "10.0.0.50/12345" {
					t.Errorf("src = %v, want 10.0.0.50/12345", event["src"])
				}
			},
		},
		{
			name:    "CEF format",
			message: `CEF:0|Vendor|Product|1.0|100|Test Event|5|src=10.0.0.1 dst=10.0.0.2 spt=1234 dpt=443 act=allow`,
			check: func(t *testing.T, event map[string]interface{}) {
				if event["src"] != "10.0.0.1" {
					t.Errorf("src = %v, want 10.0.0.1", event["src"])
				}
				if event["act"] != "allow" {
					t.Errorf("act = %v, want allow", event["act"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := map[string]interface{}{
				"message": tt.message,
			}

			err := parser.Parse(event)
			if err != nil {
				t.Errorf("Parse() error = %v", err)
				return
			}

			tt.check(t, event)
		})
	}
}
