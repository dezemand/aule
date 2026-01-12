package wssubscriptions

import (
	"encoding/json"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	eventsws "github.com/dezemandje/aule/internal/model/events/ws"
)

// SubscriptionItem defines how to create and initialize a subscription type.
type SubscriptionItem interface {
	// CreateSubscription creates a new subscription from a client request.
	CreateSubscription(client *wsproto.Client, query json.RawMessage) (Subscription, error)

	// OnInitial returns the initial data event for a newly created subscription.
	// Return nil to send nothing.
	OnInitial(sub Subscription) *eventsws.OutgoingEvent
}
