//go:build sds

package sds

import (
	sds "github.com/DataDog/sds-go-bindings"
)

// TODO(remy):
type RulesConfig struct {
	Rules []RuleConfig `json:"rules"`
}

// TODO(remy): use the actual schema
type RuleConfig struct {
	Id                 string              `json:"id"`
	Name               string              `json:"name"`
	Description        string              `json:"description"`
	Pattern            string              `json:"pattern"`
	Tags               []string            `json:"tags"`
	MatchAction        sds.MatchActionType `json:"match_action"`
	ReplacePlaceholder string              `json:"replace_placeholder"`
	Enabled            bool                `json:"enabled"`
}

// GetById returns a RuleConfig from the in-memory definitions.
// If no definitions have been received or if the rule does not exist,
// returns nil.
// This method is NOT thread safe, caller has to ensure the thread safety.
func (r RulesConfig) GetById(id string) *RuleConfig {
	for i, rc := range r.Rules {
		if rc.Id == id {
			return &r.Rules[i]
		}
	}
	return nil
}

// OnlyEnabled returns a new RulesConfig object containing only enabled rules.
// Use this to filter out disabled rules.
func (r RulesConfig) OnlyEnabled() RulesConfig {
	rules := []RuleConfig{}
	for _, rule := range r.Rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	return RulesConfig{
		Rules: rules,
	}
}
