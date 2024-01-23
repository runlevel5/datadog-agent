package netpath

import (
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"gopkg.in/yaml.v2"
)

// InstanceConfig is used to deserialize integration instance config
type InstanceConfig struct {
	DestName            string `yaml:"name"`
	DestHostname        string `yaml:"hostname"`
	Port                int    `yaml:"port"`
	FakeEventMultiplier int    `yaml:"fake_event_multiplier"`
}

type CheckConfig struct {
	DestHostname        string
	DestName            string
	FakeEventMultiplier int
	Port                int
}

// NewCheckConfig builds a new check config
func NewCheckConfig(rawInstance integration.Data, rawInitConfig integration.Data) (*CheckConfig, error) {
	instance := InstanceConfig{}

	err := yaml.Unmarshal(rawInstance, &instance)
	if err != nil {
		return nil, err
	}

	c := &CheckConfig{}

	log.Debugf("rawInstance: %s", string(rawInstance))
	c.DestHostname = instance.DestHostname
	c.DestName = instance.DestName
	c.Port = instance.Port
	c.FakeEventMultiplier = instance.FakeEventMultiplier

	if c.FakeEventMultiplier == 0 {
		c.FakeEventMultiplier = 1
	}

	log.Debugf("CheckConfig: %+v", c)
	return c, nil
}
