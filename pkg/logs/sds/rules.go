package sds

// TODO(remy):
type RulesConfig struct {
	Id          string       `json:"id"`
	Rules       []RuleConfig `json:"rules"`
	Priority    int          `json:"priority"`
	Description string       `json:"description"`
}

type MatchAction struct {
	Type           string `json:"type"`
	Placeholder    string `json:"placeholder"`
	Direction      string `json:"direction"`
	CharacterCount int    `json:"character_count"`
}

// TODO(remy): use the actual schema
type RuleConfig struct {
	Id             string      `json:"id"`
	StandardRuleId string      `json:"standard_rule_id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Pattern        string      `json:"pattern"`
	Tags           []string    `json:"tags"`
	MatchAction    MatchAction `json:"match_action"`
	IsEnabled      bool        `json:"is_enabled"`
}

// GetById returns a RuleConfig from the in-memory definitions.
// If no definitions have been received or if the rule does not exist,
// returns nil.
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
		if rule.IsEnabled {
			rules = append(rules, rule)
		}
	}
	return RulesConfig{
		Rules: rules,
	}
}
