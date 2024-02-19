package sds

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/DataDog/datadog-agent/pkg/util/log"
	sds "github.com/DataDog/sds-go-bindings"
)

type reconfigureOrderType string

const (
	// Definitions triggers the storage of a new set of standard rules
	// and reconfigure the internal SDS scanner with an existing user
	// configuration if any.
	Definitions reconfigureOrderType = "definitions"
	// UserConfig triggers a reconfiguration of the SDS scanner.
	UserConfig reconfigureOrderType = "user_config"
)

// ReconfigureOrder are used to trigger a reconfiguration
// of the SDS scanner.
type ReconfigureOrder struct {
	Type   reconfigureOrderType
	Config []byte
}

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
}

// Scanner wraps an SDS Scanner implementation, adds reconfiguration
// capabilities and telemetry on top of it.
// Most of Scanner methods are not thread safe for performance reasons, the caller
// has to ensure of the thread safety.
type Scanner struct {
	*sds.Scanner
	sync.Mutex

	definitions RulesConfig
	// rawConfig is the raw config previously received through RC.
	rawConfig []byte // XXX(remy): type
	// configuredRules are stored on configuration to retrieve rules
	// information on match. Use this read-only.
	configuredRules []RuleConfig
}

// CreateScanner creates an SDS scanner with the given raw config.
// This raw config has been either just received through RC or stored for
// an internal reconfiguration at runtime.
func CreateScanner(rawConfig []byte) (*Scanner, error) {
	scanner := &Scanner{}
	// TODO(remy): reload definitions from disk?
	order := ReconfigureOrder{
		Config: rawConfig,
		Type:   UserConfig,
	}
	err := scanner.Reconfigure(order)
	log.Debugf("creating a new SDS scanner (internal id: %p)", scanner)
	return scanner, err

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

// Reconfigure uses the given `ReconfigureOrder` to reconfigure in-memory
// definitions or user configuration.
// The order contains both the kind of reconfiguration to do and the raw bytes
// to apply the reconfiguration.
// When receiving definitions, user configuration are reloaded and scanners are
// recreated to use the newly received definitions.
// This method is thread safe, a scan can't happen at the same time.
func (s *Scanner) Reconfigure(order ReconfigureOrder) error {
	s.Lock()
	defer s.Unlock()

	if s == nil {
		log.Warn("Trying to reconfigure a nil Scanner")
		return nil
	}

	log.Debugf("reconfiguring SDS scanner (internal id: %p)", s)

	switch order.Type {
	case Definitions:
		err := s.reconfigureDefinitions(order.Config)
		// if we already received a configuration,
		// reapply it now that the definitions have changed.
		if s.rawConfig != nil {
			if rerr := s.reconfigureRules(s.rawConfig); rerr != nil {
				log.Error("Can't reconfigure SDS after having received definitions:", rerr)
				s.rawConfig = nil // we drop this configuration because it is unusable
				if err == nil {
					err = rerr
				}
			}
		}
		return err
	case UserConfig:
		return s.reconfigureRules(order.Config)
	}

	return fmt.Errorf("Scanner.Reconfigure: Unknown order type: %v", order.Type)
}

// reconfigureDefinitions stores in-memory definitions received through RC.
// This is NOT reconfiguring the internal SDS scanner, call `reconfigureRules`
// if you have to.
// This method is NOT thread safe, the caller has to ensure the thread safety.
func (s *Scanner) reconfigureDefinitions(rawConfig []byte) error {
	if rawConfig == nil {
		return fmt.Errorf("Invalid nil raw configuration for definitions")
	}

	var config RulesConfig
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return fmt.Errorf("Can't unmarshal raw configuration: %v", err)
	}

	s.definitions = config
	log.Info("Reconfigured SDS definitions.")
	return nil
}

// reconfigureRules reconfigures the internal SDS scanner using the in-memory
// definitions. Could possibly delete and recreate the internal SDS scanner if
// necessary.
// This method is NOT thread safe, caller has to ensure the thread safety.
func (s *Scanner) reconfigureRules(rawConfig []byte) error {
	if s.definitions.Rules == nil {
		return fmt.Errorf("Received an user configuration before receiving SDS rules definitions")
	}

	if rawConfig == nil {
		return fmt.Errorf("Invalid nil raw configuration received for user configuration")
	}

	var config RulesConfig
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return fmt.Errorf("Can't unmarshal raw configuration: %v", err)
	}

	// if we received an empty array of rules, interprets this as "stop SDS".
	if len(config.Rules) == 0 {
		log.Info("Received an empty configuration, stopping the SDS scanner.")
		// destroy the old scanner
		if s.Scanner != nil {
			s.Scanner.Delete()
			s.Scanner = nil
			s.rawConfig = rawConfig
			s.configuredRules = nil
			return nil
		}
	}

	// prepare the scanner rules
	var sdsRules []sds.Rule
	for _, rule := range config.Rules {
		// TODO(remy): other type of configuration?
		switch rule.MatchAction {
		case sds.MatchActionRedact:
			sdsRules = append(sdsRules, sds.NewRedactingRule(rule.Name, rule.Pattern, rule.ReplacePlaceholder))
			//		case sds.MatchActionHash:
			//    		sdsRules = append(sdsRules, sds.NewHashRule(rule.Name, rule.Pattern, rule.ReplacePlaceholder))
		default:
			log.Warnf("Unknown MatchAction type (%v) for rule '%s':", rule.MatchAction, rule.Name)
		}
	}
	// create the new SDS Scanner
	var scanner *sds.Scanner
	var err error

	if scanner, err = sds.CreateScanner(sdsRules); err != nil {
		return fmt.Errorf("while configuration an SDS Scanner: %v", err)
	}

	// destroy the old scanner
	if s.Scanner != nil {
		s.Scanner.Delete()
		s.Scanner = nil
	}

	// store the raw configuration for a later refresh
	// if we receive new definitions
	s.rawConfig = rawConfig
	s.configuredRules = config.Rules

	log.Info("Created an SDS scanner with", len(scanner.Rules), "rules")
	s.Scanner = scanner

	return nil
}

// Scan scans the given `event` using the internal SDS scanner.
// Returns an error if the internal SDS scanner is not ready. If you need to
// validate that the internal SDS scanner can be used, use `IsReady()`.
// This method is thread safe, a reconfiguration can't happen at the same time.
func (s *Scanner) Scan(event []byte) ([]byte, []sds.RuleMatch, error) {
	s.Lock()
	defer s.Unlock()
	// TODO(remy): telemetry

	if s.Scanner == nil {
		return nil, nil, fmt.Errorf("can't Scan with an unitialized scanner")
	}

	return s.Scanner.Scan(event)
}

func (s *Scanner) GetRuleByIdx(idx uint32) (RuleConfig, error) {
	if s.Scanner == nil {
		return RuleConfig{}, fmt.Errorf("scanner not configured")
	}
	if uint32(len(s.configuredRules)) <= idx {
		return RuleConfig{}, fmt.Errorf("scanner not containing enough rules")
	}
	return s.configuredRules[idx], nil
}

// Delete deallocates the internal SDS scanner.
// This method is NOT thread safe, caller has to ensure the thread safety.
func (s *Scanner) Delete() {
	if s.Scanner != nil {
		s.Scanner.Delete()
		s.rawConfig = nil
		s.configuredRules = nil
	}
	s.Scanner = nil
}

// IsReady returns true if this Scanner can be used
// to scan events and that at least one rule would be applied.
// This method is NOT thread safe, caller has to ensure the thread safety.
func (s *Scanner) IsReady() bool {
	if s == nil {
		return false
	}
	if s.Scanner == nil {
		return false
	}
	if len(s.Scanner.Rules) == 0 {
		return false
	}

	return true
}
