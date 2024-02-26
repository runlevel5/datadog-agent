//go:build sds

package sds

import (
	"bytes"
	"testing"
)

func TestCreateScanner(t *testing.T) {
	json := []byte(`
        {"rules":[
            {
                "id":"zero-0",
                "description":"zero desc",
                "name":"zero",
                "pattern":"zero",
                "replace_placeholder":"[redacted]",
                "match_action":"Redact",
                "enabled":false
            },{
                "id":"one-1",
                "description":"one desc",
                "name":"one",
                "pattern":"one",
                "match_action":"Hash",
                "enabled":false
            },{
                "id":"two-2",
                "description":"two desc",
                "name":"two",
                "pattern":"two",
                "match_action":"Redact",
                "enabled":false
            }
        ]}
    `)

	// scanner creation
	// -----
	s, err := CreateScanner(json)

	// should fail because we did not receive any definitions yet
	if err == nil {
		t.Errorf("creating this scanner should fail because no definitions are set")
	}
	if s == nil {
		t.Error("the scanner should not be nil after a creation")
	}

	err = s.Reconfigure(ReconfigureOrder{
		Type:   Definitions,
		Config: json,
	})

	if err != nil {
		t.Errorf("configuring the definitions should not fail: %v", err)
	}

	// now that we have some definitions, we can configure the scanner
	err = s.Reconfigure(ReconfigureOrder{
		Type:   UserConfig,
		Config: json,
	})

	if err == nil {
		t.Errorf("this one should fail since all rules are disabled")
	}

	// enable 2 of the 3 rules
	// ------

	json = bytes.Replace(json, []byte("\"enabled\":false"), []byte("\"enabled\":true"), 2)

	err = s.Reconfigure(ReconfigureOrder{
		Type:   UserConfig,
		Config: json,
	})

	if err != nil {
		t.Errorf("this one should not fail since two rules are enabled: %v", err)
	}

	if len(s.configuredRules) != 2 {
		t.Errorf("only two rules should be part of this scanner. len == %d", len(s.configuredRules))
	}

	// order matters, it's ok to test rules by [] access
	if s.configuredRules[0].Name != "zero" {
		t.Error("incorrect rules selected for configuration")
	}
	if s.configuredRules[1].Name != "one" {
		t.Error("incorrect rules selected for configuration")
	}

	// compare rules returned by GetRuleByIdx

	zero, err := s.GetRuleByIdx(0)
	if err != nil {
		t.Errorf("GetRuleByIdx on 0 should not fail: %v", err)
	}
	if s.configuredRules[0].Id != zero.Id {
		t.Error("incorrect rule returned")
	}

	one, err := s.GetRuleByIdx(1)
	if err != nil {
		t.Errorf("GetRuleByIdx on 1 should not fail: %v", err)
	}
	if s.configuredRules[1].Id != one.Id {
		t.Error("incorrect rule returned")
	}

	// disable the rule zero
	// only "one" is left enabled
	// -----

	json = bytes.Replace(json, []byte("\"enabled\":true"), []byte("\"enabled\":false"), 1)

	err = s.Reconfigure(ReconfigureOrder{
		Type:   UserConfig,
		Config: json,
	})

	if err != nil {
		t.Errorf("this one should not fail since one rule is enabled: %v", err)
	}

	if len(s.configuredRules) != 1 {
		t.Errorf("only one rules should be part of this scanner. len == %d", len(s.configuredRules))
	}

	// order matters, it's ok to test rules by [] access
	if s.configuredRules[0].Name != "one" {
		t.Error("incorrect rule selected for configuration")
	}

	rule, err := s.GetRuleByIdx(0)
	if err != nil {
		t.Error("incorrect rule returned")
	}
	if rule.Id != s.configuredRules[0].Id || rule.Name != "one" {
		t.Error("the scanner hasn't been configured with the good rule")
	}
}

func TestIsReady(t *testing.T) {
	json := []byte(`
        {"rules":[
            {
                "id":"zero-0",
                "description":"zero desc",
                "name":"zero",
                "pattern":"zero",
                "replace_placeholder":"[redacted]",
                "match_action":"Redact",
                "enabled":true
            },{
                "id":"one-1",
                "description":"one desc",
                "name":"one",
                "pattern":"one",
                "match_action":"Hash",
                "enabled":true
            }        ]}
    `)

	// scanner creation
	// -----

	s, err := CreateScanner(json)

	// should fail because we did not receive any definitions yet
	if err == nil {
		t.Errorf("creating this scanner should fail because no definitions are set")
	}
	if s == nil {
		t.Error("the scanner should not be nil after a creation")
	}

	if s.IsReady() != false {
		t.Error("at this stage, the scanner should not be considered ready, no definitions received")
	}

	err = s.Reconfigure(ReconfigureOrder{
		Type:   Definitions,
		Config: json,
	})

	if err != nil {
		t.Errorf("configuring the definitions should not fail: %v", err)
	}

	if s.IsReady() != false {
		t.Error("at this stage, the scanner should not be considered ready, no user config received")
	}

	// now that we have some definitions, we can configure the scanner
	err = s.Reconfigure(ReconfigureOrder{
		Type:   UserConfig,
		Config: json,
	})

	if s.IsReady() != true {
		t.Error("at this stage, the scanner should be considered ready")
	}
}

// TestScan validates that everything fits and works. It's not validating
// the scanning feature itself, which is done in the library.
func TestScan(t *testing.T) {
	json := []byte(`
        {"rules":[
            {
                "id":"zero-0",
                "description":"zero desc",
                "name":"zero",
                "pattern":"zero",
                "replace_placeholder":"[redacted]",
                "match_action":"Redact",
                "enabled":true
            },{
                "id":"one-1",
                "description":"one desc",
                "name":"one",
                "pattern":"one",
                "match_action":"Redact",
                "replace_placeholder":"[REDACTED]",
                "enabled":true
            }
        ]}
    `)

	// scanner creation
	// -----

	s, _ := CreateScanner(json)
	if s == nil {
		t.Error("the returned scanner should not be nil")
	}
	_ = s.Reconfigure(ReconfigureOrder{
		Type:   Definitions,
		Config: json,
	})
	_ = s.Reconfigure(ReconfigureOrder{
		Type:   UserConfig,
		Config: json,
	})

	if s.IsReady() != true {
		t.Error("at this stage, the scanner should be considered ready")
	}

	type result struct {
		event      string
		matchCount int
	}

	tests := map[string]result{
		"one two three go!": {
			event:      "[REDACTED] two three go!",
			matchCount: 1,
		},
		"after zero comes one, after one comes two, and the rest is history": {
			event:      "after [redacted] comes [REDACTED], after [REDACTED] comes two, and the rest is history",
			matchCount: 3,
		},
	}

	for k, v := range tests {
		processed, rulesMatch, err := s.Scan([]byte(k))
		if err != nil {
			t.Errorf("scanning these event should not fail. err received: %v", err)
		}
		if processed == nil {
			t.Errorf("incorrect result: nil processed event returned")
		}
		if string(processed) != v.event {
			t.Errorf("incorrect result, expected \"%v\" got \"%v\"", v.event, string(processed))
		}
		if len(rulesMatch) != v.matchCount {
			t.Errorf("incorrect result, expected %d matches, got %d", v.matchCount, len(rulesMatch))
		}
	}
}
