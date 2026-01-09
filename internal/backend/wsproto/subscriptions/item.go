package wssubscriptions

import (
	"encoding/json"

	"github.com/dezemandje/aule/internal/backend/wsproto"
)

type SubscriptionItem interface {
	CreateSubscription(client *wsproto.Client, query json.RawMessage) (Subscription, error)

	OnInitial(c wsproto.Ctx) error
}
