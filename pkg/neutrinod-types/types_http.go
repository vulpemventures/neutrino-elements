package neutrinodtypes

import "github.com/vulpemventures/neutrino-elements/pkg/scanner"

type SubscriptionRequestHttp struct {
	ActionType       ActionType
	EventTypes       []scanner.EventType `json:"eventTypes"`
	DescriptorWallet string              `json:"descriptorWallet"`
	StartBlockHeight int                 `json:"startBlockHeight"`
	EndpointUrl      string              `json:"endpointUrl"`
}
