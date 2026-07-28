// Package skillhubdbootstrap provides the immutable-at-startup configuration
// consumed by the daemon's framework Initiator.
package skillhubdbootstrap

import "sync"

type Config struct {
	Host      string
	Port      int
	SecretKey string
}

var current = struct {
	sync.RWMutex
	config Config
	set    bool
}{}

func Configure(config Config) {
	current.Lock()
	defer current.Unlock()
	current.config = config
	current.set = true
}

func Current() (Config, bool) {
	current.RLock()
	defer current.RUnlock()
	return current.config, current.set
}
