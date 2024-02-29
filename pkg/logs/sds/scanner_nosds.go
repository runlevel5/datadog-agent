//go:build !sds

package sds

import (
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

// Scanner mock.
type Scanner struct {
}

// Match mock.
type Match struct {
	RuleIdx uint32
}

// CreateScanner creates a scanner for unsupported platforms/architectures.
func CreateScanner(rawConfig []byte) (*Scanner, error) {
	return nil, nil
}

// Reconfigure mocks the Reconfigure function.
func (s *Scanner) Reconfigure(order ReconfigureOrder) error {
	return nil
}

// Delete mocks the Delete function.
func (s *Scanner) Delete() {}

// GetRuleByIdx mocks the GetRuleByIdx function.
func (s *Scanner) GetRuleByIdx(_ uint32) (RuleConfig, error) {
	return RuleConfig{}, nil
}

// IsReady mocks the IsReady function.
func (s *Scanner) IsReady() bool { return false }

// Scan mocks the Scan function.
func (s *Scanner) Scan(_ []byte, msg *message.Message) (bool, []byte, error) {
	return false, nil, nil
}
