package sds

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
