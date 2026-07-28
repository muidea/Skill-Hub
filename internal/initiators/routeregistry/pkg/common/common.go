package common

import enginehttp "github.com/muidea/magicEngine/http"

const RouteRegistryInitiator = "skillhub.initiator.routeregistry"

// RouteRegistryHelper exposes route declaration only. Application Modules do
// not receive the listener or http.Server lifecycle.
type RouteRegistryHelper interface {
	GetRouteRegistry() enginehttp.RouteRegistry
}

// GatewayRuntimeHelper is reserved for the daemon process service, which owns
// activation and waits for the listener to finish.
type GatewayRuntimeHelper interface {
	RouteRegistryHelper
	Start() error
	Done() <-chan error
}
