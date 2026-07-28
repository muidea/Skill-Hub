package hubclient

import "github.com/muidea/skill-hub/internal/adapters/hubclient/service"

// HubClient is the HTTP protocol adapter. The public adapter surface is the
// protocol client itself; there is no additional service facade.
type HubClient = service.Client

func New(baseURL string) *HubClient {
	return service.New(baseURL)
}

func NewWithSecret(baseURL, secretKey string) *HubClient {
	return service.NewWithSecret(baseURL, secretKey)
}
