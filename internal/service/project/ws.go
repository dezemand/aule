package projectsservice

import (
	"encoding/json"
	"time"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	wssubscriptions "github.com/dezemandje/aule/internal/backend/wsproto/subscriptions"
	"github.com/dezemandje/aule/internal/event"
	"github.com/dezemandje/aule/internal/eventhandler"
)

type WsHandler struct {
	service       *Service
	eventHandler  eventhandler.EventHandler
	subscriptions *wssubscriptions.Service
}

func NewWsHandler(
	eventHandler eventhandler.EventHandler,
	subscriptions *wssubscriptions.Service,
	service *Service,
) *WsHandler {
	handler := &WsHandler{
		eventHandler:  eventHandler,
		subscriptions: subscriptions,
		service:       service,
	}
	subscriptions.Register("projects.list", &projectsSubItem{handler: handler})
	//eventHandler.Register("projects.create", &projectsEventHandler{handler: handler})

	go handler.foo()

	return handler
}

func (h *WsHandler) foo() {
	ticker := time.NewTicker(10 * time.Second)

	for {
		<-ticker.C
		h.subscriptions.SendSubscriptionEvent(
			MsgTypeProjectsList,
			func(s wssubscriptions.Subscription) bool {
				return true
			},
			h.OnListProjects,
		)
	}
}

func (h *WsHandler) OnCreateProject(c wsproto.Ctx) error {
	return nil
}

func (h *WsHandler) OnListProjects(c wsproto.Ctx) error {
	var body ProjectsListRequest
	if err := c.Body(&body); err != nil {
		return c.ReplyError("invalid_request", "could not parse request body", nil)
	}

	userID := c.Client().UserID()

	projects, memberships, err := h.service.repository.FindProjectsForUser(c.Context(), userID)
	if err != nil {
		return c.ReplyError("error", err.Error(), nil)
	}

	_ = memberships

	return c.Reply(&ProjectsListResponse{
		Projects: projects,
	})
}

func (h *WsHandler) send(evt eventhandler.Event) error {
	ev := evt.(*event.CreateProjectEvent)

	return h.subscriptions.SendSubscriptionEvent(
		MsgTypeProjectsList,
		func(s wssubscriptions.Subscription) bool {
			return s.UserID() == ev.CreatorID
		},
		h.OnListProjects,
	)
}

type projectsSubItem struct {
	handler *WsHandler
}

func (p *projectsSubItem) CreateSubscription(client *wsproto.Client, query json.RawMessage) (wssubscriptions.Subscription, error) {
	return wssubscriptions.NewSubscription(client, MsgTypeProjectsList, nil), nil
}

func (p *projectsSubItem) OnInitial(c wsproto.Ctx) error {
	return p.handler.OnListProjects(c)
}

type projectsEventHandler struct {
	handler *WsHandler
}

func (p *projectsEventHandler) Handle(evt eventhandler.Event) error {
	return p.handler.send(evt)
}
