package types

import "time"

// Config is the configuration structure
type Config struct {
	BootstrapNodes []string
	ServerID string
	ServerPublicKey string
	ServerPrivateKey string
	HTTPListenPort int
	RetryCount int
	RetryInterval time.Duration
}
