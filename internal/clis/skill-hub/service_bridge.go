package cli

import remote "github.com/muidea/skill-hub/internal/runtime/remote"

// serviceBridgeClient is retained as a CLI-local alias so command handlers and
// their fakes share one typed remote contract. Service discovery itself lives
// in internal/runtime/remote.
type serviceBridgeClient = remote.Client

var serviceBridgeResolver = remote.Resolve

func hubClientIfAvailable() (serviceBridgeClient, bool) {
	return serviceBridgeResolver()
}
