package projectsservice

import (
	"context"
	"encoding/json"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	wssubscriptions "github.com/dezemandje/aule/internal/backend/wsproto/subscriptions"
	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/event"
	eventsprojects "github.com/dezemandje/aule/internal/model/events/projects"
	eventsws "github.com/dezemandje/aule/internal/model/events/ws"
	modelsws "github.com/dezemandje/aule/internal/model/ws"
	"github.com/google/uuid"
)

// Internal event topics for WS message handling
var (
	topicCreateProject = event.NewTopic[createProjectEvent]("ws.projects.create")
	topicUpdateProject = event.NewTopic[updateProjectEvent]("ws.projects.update")
	topicDeleteProject = event.NewTopic[deleteProjectEvent]("ws.projects.delete")
	topicGetProject    = event.NewTopic[getProjectEvent]("ws.projects.get")
	topicListProjects  = event.NewTopic[listProjectsEvent]("ws.projects.list")
	topicListMembers   = event.NewTopic[listMembersEvent]("ws.projects.members.list")
	topicAddMember     = event.NewTopic[addMemberEvent]("ws.projects.members.add")
	topicUpdateMember  = event.NewTopic[updateMemberEvent]("ws.projects.members.update")
	topicRemoveMember  = event.NewTopic[removeMemberEvent]("ws.projects.members.remove")
)

// Internal event types for WS message handling
type createProjectEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	Name         string
	Description  string
}

type updateProjectEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	ProjectID    string
	Name         *string
	Description  *string
	Status       *domain.ProjectStatus
	Purpose      *domain.ProjectPurpose
	Scope        *domain.ProjectScope
	Governance   *domain.ProjectGovernance
	TaskConfig   *domain.ProjectTaskConfig
	AgentConfig  *domain.ProjectAgentConfig
}

type deleteProjectEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	ProjectID    string
}

type getProjectEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	ProjectID    string
}

type listProjectsEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
}

type listMembersEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	ProjectID    string
}

type addMemberEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	ProjectID    string
	MemberUserID string
	Role         domain.ProjectMemberRole
	Permissions  *domain.ProjectMemberPermissions
}

type updateMemberEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	ProjectID    string
	MemberUserID string
	Role         domain.ProjectMemberRole
	Permissions  *domain.ProjectMemberPermissions
}

type removeMemberEvent struct {
	ClientID     uuid.UUID
	UserID       domain.UserID
	RequestMsgID uuid.UUID
	ProjectID    string
	MemberUserID string
}

// Handler handles project-related WebSocket messages via the event bus.
type Handler struct {
	bus           *event.Bus
	service       *Service
	subscriptions *wssubscriptions.Service
	subs          []event.Subscription
}

// NewHandler creates a new project handler.
func NewHandler(bus *event.Bus, service *Service, subscriptions *wssubscriptions.Service) *Handler {
	h := &Handler{
		bus:           bus,
		service:       service,
		subscriptions: subscriptions,
	}

	// Register subscription types
	subscriptions.Register("projects.list", &projectsListSubItem{handler: h})
	subscriptions.Register("projects.detail", &projectsDetailSubItem{handler: h})
	subscriptions.Register("projects.members", &projectsMembersSubItem{handler: h})

	return h
}

// SetupEventHandlers registers all event handlers for project operations.
func (h *Handler) SetupEventHandlers() {
	h.subs = []event.Subscription{
		// Transform incoming WS messages to internal events
		wsproto.WsToEvent(h.bus, modelsws.MsgTypeProjectCreate, topicCreateProject,
			func(payload modelsws.ProjectCreateRequest, evt event.Event[eventsws.IncomingEvent]) createProjectEvent {
				return createProjectEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					Name:         payload.Name,
					Description:  payload.Description,
				}
			}),

		wsproto.WsToEvent(h.bus, modelsws.MsgTypeProjectUpdate, topicUpdateProject,
			func(payload modelsws.ProjectUpdateRequest, evt event.Event[eventsws.IncomingEvent]) updateProjectEvent {
				return updateProjectEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					ProjectID:    payload.ID,
					Name:         payload.Name,
					Description:  payload.Description,
					Status:       payload.Status,
					Purpose:      payload.Purpose,
					Scope:        payload.Scope,
					Governance:   payload.Governance,
					TaskConfig:   payload.TaskConfig,
					AgentConfig:  payload.AgentConfig,
				}
			}),

		wsproto.WsToEvent(h.bus, modelsws.MsgTypeProjectDelete, topicDeleteProject,
			func(payload modelsws.ProjectDeleteRequest, evt event.Event[eventsws.IncomingEvent]) deleteProjectEvent {
				return deleteProjectEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					ProjectID:    payload.ID,
				}
			}),

		wsproto.WsToEvent(h.bus, modelsws.MsgTypeProjectGet, topicGetProject,
			func(payload modelsws.ProjectGetRequest, evt event.Event[eventsws.IncomingEvent]) getProjectEvent {
				return getProjectEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					ProjectID:    payload.ID,
				}
			}),

		wsproto.WsToEvent(h.bus, modelsws.MsgTypeProjectsListReq, topicListProjects,
			func(payload modelsws.ProjectsListRequest, evt event.Event[eventsws.IncomingEvent]) listProjectsEvent {
				return listProjectsEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
				}
			}),

		// Member management WS to event transformers
		wsproto.WsToEvent(h.bus, modelsws.MsgTypeMembersListReq, topicListMembers,
			func(payload modelsws.MembersListRequest, evt event.Event[eventsws.IncomingEvent]) listMembersEvent {
				return listMembersEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					ProjectID:    payload.ProjectID,
				}
			}),

		wsproto.WsToEvent(h.bus, modelsws.MsgTypeMemberAdd, topicAddMember,
			func(payload modelsws.MemberAddRequest, evt event.Event[eventsws.IncomingEvent]) addMemberEvent {
				return addMemberEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					ProjectID:    payload.ProjectID,
					MemberUserID: payload.UserID,
					Role:         payload.Role,
					Permissions:  payload.Permissions,
				}
			}),

		wsproto.WsToEvent(h.bus, modelsws.MsgTypeMemberUpdate, topicUpdateMember,
			func(payload modelsws.MemberUpdateRequest, evt event.Event[eventsws.IncomingEvent]) updateMemberEvent {
				return updateMemberEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					ProjectID:    payload.ProjectID,
					MemberUserID: payload.UserID,
					Role:         payload.Role,
					Permissions:  payload.Permissions,
				}
			}),

		wsproto.WsToEvent(h.bus, modelsws.MsgTypeMemberRemove, topicRemoveMember,
			func(payload modelsws.MemberRemoveRequest, evt event.Event[eventsws.IncomingEvent]) removeMemberEvent {
				return removeMemberEvent{
					ClientID:     evt.Payload().ClientID,
					UserID:       evt.Payload().UserID,
					RequestMsgID: evt.Payload().Message.MessageID,
					ProjectID:    payload.ProjectID,
					MemberUserID: payload.UserID,
				}
			}),

		// Handle internal events (call service methods)
		event.Subscribe(h.bus, topicCreateProject, h.handleCreateProject),
		event.Subscribe(h.bus, topicUpdateProject, h.handleUpdateProject),
		event.Subscribe(h.bus, topicDeleteProject, h.handleDeleteProject),
		event.Subscribe(h.bus, topicGetProject, h.handleGetProject),
		event.Subscribe(h.bus, topicListProjects, h.handleListProjects),
		event.Subscribe(h.bus, topicListMembers, h.handleListMembers),
		event.Subscribe(h.bus, topicAddMember, h.handleAddMember),
		event.Subscribe(h.bus, topicUpdateMember, h.handleUpdateMember),
		event.Subscribe(h.bus, topicRemoveMember, h.handleRemoveMember),

		// Bridge domain events to WS subscriptions for real-time updates
		wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicProjectCreated, "projects.list",
			func(ctx context.Context, e event.Event[eventsprojects.ProjectCreatedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
				return func(sub wssubscriptions.Subscription) bool {
						// Notify the creator's subscriptions
						return sub.UserID() == e.Payload().CreatorID
					}, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
						return h.buildProjectsListEvent(sub)
					}
			}),

		wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicProjectUpdated, "projects.list",
			func(ctx context.Context, e event.Event[eventsprojects.ProjectUpdatedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
				return func(sub wssubscriptions.Subscription) bool {
						// Notify all users who have subscribed to project list
						// In a real app, you'd check if user has access to this project
						return true
					}, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
						return h.buildProjectsListEvent(sub)
					}
			}),

		wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicProjectDeleted, "projects.list",
			func(ctx context.Context, e event.Event[eventsprojects.ProjectDeletedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
				return func(sub wssubscriptions.Subscription) bool {
						return true
					}, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
						return h.buildProjectsListEvent(sub)
					}
			}),

		// Bridge domain events to projects.detail subscriptions
		wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicProjectUpdated, "projects.detail",
			func(ctx context.Context, e event.Event[eventsprojects.ProjectUpdatedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
				project := e.Payload().Project // Copy to take address
				return func(sub wssubscriptions.Subscription) bool {
						// Only notify subscriptions for this specific project
						query, ok := sub.Query().(*projectDetailQuery)
						if !ok {
							return false
						}
						return query.ProjectID == e.Payload().ProjectID
					}, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
						return h.buildProjectDetailEvent(sub, &project)
					}
			}),

		wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicProjectDeleted, "projects.detail",
			func(ctx context.Context, e event.Event[eventsprojects.ProjectDeletedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
				return func(sub wssubscriptions.Subscription) bool {
						// Notify subscriptions for the deleted project
						query, ok := sub.Query().(*projectDetailQuery)
						if !ok {
							return false
						}
						return query.ProjectID == e.Payload().ProjectID
					}, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
						// Send a deleted notification
						return h.buildProjectDeletedEvent(sub, e.Payload().ProjectID)
					}
			}),

		// Bridge member events to projects.members subscriptions
		wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicMemberAdded, "projects.members",
			func(ctx context.Context, e event.Event[eventsprojects.MemberAddedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
				return func(sub wssubscriptions.Subscription) bool {
						query, ok := sub.Query().(*projectMembersQuery)
						if !ok {
							return false
						}
						return query.ProjectID == e.Payload().ProjectID
					}, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
						query := sub.Query().(*projectMembersQuery)
						return h.buildMembersListEvent(sub, query.ProjectID)
					}
			}),

		wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicMemberUpdated, "projects.members",
			func(ctx context.Context, e event.Event[eventsprojects.MemberUpdatedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
				return func(sub wssubscriptions.Subscription) bool {
						query, ok := sub.Query().(*projectMembersQuery)
						if !ok {
							return false
						}
						return query.ProjectID == e.Payload().ProjectID
					}, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
						query := sub.Query().(*projectMembersQuery)
						return h.buildMembersListEvent(sub, query.ProjectID)
					}
			}),

		wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicMemberRemoved, "projects.members",
			func(ctx context.Context, e event.Event[eventsprojects.MemberRemovedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
				return func(sub wssubscriptions.Subscription) bool {
						query, ok := sub.Query().(*projectMembersQuery)
						if !ok {
							return false
						}
						return query.ProjectID == e.Payload().ProjectID
					}, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
						query := sub.Query().(*projectMembersQuery)
						return h.buildMembersListEvent(sub, query.ProjectID)
					}
			}),
	}
}

// Cleanup unsubscribes all event handlers.
func (h *Handler) Cleanup() {
	for _, sub := range h.subs {
		if sub != nil {
			sub.Unsubscribe()
		}
	}
}

// buildProjectsListEvent builds an outgoing event with the projects list for a subscription.
func (h *Handler) buildProjectsListEvent(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
	projects, err := h.service.ListProjects(context.Background(), sub.UserID())
	if err != nil {
		// Return error event
		payload, _ := json.Marshal(modelsws.ErrorPayload{
			Code:    "list_projects_failed",
			Message: err.Error(),
		})
		subID := sub.ID()
		return &eventsws.OutgoingEvent{
			To:             []eventsws.OutgoingTo{{ID: sub.ClientID()}},
			Type:           "error",
			Payload:        payload,
			SubscriptionID: &subID,
		}
	}

	payload, err := json.Marshal(&modelsws.ProjectsListResponse{Projects: projects})
	if err != nil {
		return nil
	}

	subID := sub.ID()
	return &eventsws.OutgoingEvent{
		To:             []eventsws.OutgoingTo{{ID: sub.ClientID()}},
		Type:           modelsws.MsgTypeProjectsList,
		Payload:        payload,
		SubscriptionID: &subID,
	}
}

// sendResponse sends a response to a specific client.
func (h *Handler) sendResponse(clientID uuid.UUID, replyTo uuid.UUID, typ string, payload any) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	event.Publish(h.bus, eventsws.TopicOutgoing.Event(eventsws.OutgoingEvent{
		To:      []eventsws.OutgoingTo{{ID: clientID}},
		Type:    typ,
		Payload: payloadBytes,
		ReplyTo: &replyTo,
	}))
}

// sendError sends an error response to a specific client.
func (h *Handler) sendError(clientID uuid.UUID, replyTo uuid.UUID, code, message string) {
	payloadBytes, _ := json.Marshal(modelsws.ErrorPayload{
		Code:    code,
		Message: message,
	})

	event.Publish(h.bus, eventsws.TopicOutgoing.Event(eventsws.OutgoingEvent{
		To:      []eventsws.OutgoingTo{{ID: clientID}},
		Type:    "error",
		Payload: payloadBytes,
		ReplyTo: &replyTo,
	}))
}

func (h *Handler) handleCreateProject(ctx context.Context, evt event.Event[createProjectEvent]) error {
	payload := evt.Payload()

	project, err := h.service.CreateProject(ctx, payload.UserID, payload.Name, payload.Description)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "create_project_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeProjectCreated, &modelsws.ProjectCreatedResponse{
		Project: *project,
	})
	return nil
}

func (h *Handler) handleUpdateProject(ctx context.Context, evt event.Event[updateProjectEvent]) error {
	payload := evt.Payload()

	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_project_id", "invalid project ID format")
		return nil
	}

	update := &ProjectUpdate{
		Name:        payload.Name,
		Description: payload.Description,
		Status:      payload.Status,
		Purpose:     payload.Purpose,
		Scope:       payload.Scope,
		Governance:  payload.Governance,
		TaskConfig:  payload.TaskConfig,
		AgentConfig: payload.AgentConfig,
	}

	project, err := h.service.UpdateProject(ctx, payload.UserID, domain.ProjectID(projectID), update)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "update_project_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeProjectUpdated, &modelsws.ProjectUpdatedResponse{
		Project: *project,
	})
	return nil
}

func (h *Handler) handleDeleteProject(ctx context.Context, evt event.Event[deleteProjectEvent]) error {
	payload := evt.Payload()

	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_project_id", "invalid project ID format")
		return nil
	}

	if err := h.service.DeleteProject(ctx, payload.UserID, domain.ProjectID(projectID)); err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "delete_project_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeProjectDeleted, &modelsws.ProjectDeletedResponse{
		ID: payload.ProjectID,
	})
	return nil
}

func (h *Handler) handleGetProject(ctx context.Context, evt event.Event[getProjectEvent]) error {
	payload := evt.Payload()

	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_project_id", "invalid project ID format")
		return nil
	}

	project, err := h.service.GetProject(ctx, payload.UserID, domain.ProjectID(projectID))
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "get_project_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeProject, &modelsws.ProjectResponse{
		Project: *project,
	})
	return nil
}

func (h *Handler) handleListProjects(ctx context.Context, evt event.Event[listProjectsEvent]) error {
	payload := evt.Payload()

	projects, err := h.service.ListProjects(ctx, payload.UserID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "list_projects_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeProjectsList, &modelsws.ProjectsListResponse{
		Projects: projects,
	})
	return nil
}

func (h *Handler) handleListMembers(ctx context.Context, evt event.Event[listMembersEvent]) error {
	payload := evt.Payload()

	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_project_id", "invalid project ID format")
		return nil
	}

	members, err := h.service.ListMembers(ctx, payload.UserID, domain.ProjectID(projectID))
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "list_members_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeMembersList, &modelsws.MembersListResponse{
		Members: members,
	})
	return nil
}

func (h *Handler) handleAddMember(ctx context.Context, evt event.Event[addMemberEvent]) error {
	payload := evt.Payload()

	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_project_id", "invalid project ID format")
		return nil
	}

	memberUserID, err := uuid.Parse(payload.MemberUserID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_user_id", "invalid user ID format")
		return nil
	}

	err = h.service.AddMember(ctx, payload.UserID, domain.ProjectID(projectID), domain.UserID(memberUserID), payload.Role)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "add_member_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeMemberAdded, &modelsws.MemberAddedResponse{
		ProjectID: payload.ProjectID,
		UserID:    payload.MemberUserID,
	})
	return nil
}

func (h *Handler) handleUpdateMember(ctx context.Context, evt event.Event[updateMemberEvent]) error {
	payload := evt.Payload()

	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_project_id", "invalid project ID format")
		return nil
	}

	memberUserID, err := uuid.Parse(payload.MemberUserID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_user_id", "invalid user ID format")
		return nil
	}

	err = h.service.UpdateMember(ctx, payload.UserID, domain.ProjectID(projectID), domain.UserID(memberUserID), payload.Role, payload.Permissions)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "update_member_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeMemberUpdated, &modelsws.MemberUpdatedResponse{
		ProjectID: payload.ProjectID,
		UserID:    payload.MemberUserID,
	})
	return nil
}

func (h *Handler) handleRemoveMember(ctx context.Context, evt event.Event[removeMemberEvent]) error {
	payload := evt.Payload()

	projectID, err := uuid.Parse(payload.ProjectID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_project_id", "invalid project ID format")
		return nil
	}

	memberUserID, err := uuid.Parse(payload.MemberUserID)
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "invalid_user_id", "invalid user ID format")
		return nil
	}

	err = h.service.RemoveMember(ctx, payload.UserID, domain.ProjectID(projectID), domain.UserID(memberUserID))
	if err != nil {
		h.sendError(payload.ClientID, payload.RequestMsgID, "remove_member_failed", err.Error())
		return nil
	}

	h.sendResponse(payload.ClientID, payload.RequestMsgID, modelsws.MsgTypeMemberRemoved, &modelsws.MemberRemovedResponse{
		ProjectID: payload.ProjectID,
		UserID:    payload.MemberUserID,
	})
	return nil
}

// projectsListSubItem implements SubscriptionItem for projects list subscriptions.
type projectsListSubItem struct {
	handler *Handler
}

func (p *projectsListSubItem) CreateSubscription(client *wsproto.Client, query json.RawMessage) (wssubscriptions.Subscription, error) {
	return wssubscriptions.NewSubscription(client, "projects.list", nil), nil
}

func (p *projectsListSubItem) OnInitial(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
	return p.handler.buildProjectsListEvent(sub)
}

// projectDetailQuery is the query type for projects.detail subscriptions.
type projectDetailQuery struct {
	ProjectID domain.ProjectID `json:"project_id"`
}

// projectsDetailSubItem implements SubscriptionItem for single project subscriptions.
type projectsDetailSubItem struct {
	handler *Handler
}

func (p *projectsDetailSubItem) CreateSubscription(client *wsproto.Client, query json.RawMessage) (wssubscriptions.Subscription, error) {
	var q projectDetailQuery
	if err := json.Unmarshal(query, &q); err != nil {
		return nil, err
	}

	// Validate the project ID
	if q.ProjectID == domain.ProjectID(uuid.Nil) {
		return nil, ErrProjectNotFound
	}

	return wssubscriptions.NewSubscription(client, "projects.detail", &q), nil
}

func (p *projectsDetailSubItem) OnInitial(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
	query, ok := sub.Query().(*projectDetailQuery)
	if !ok {
		return nil
	}

	project, err := p.handler.service.GetProject(context.Background(), sub.UserID(), query.ProjectID)
	if err != nil {
		payload, _ := json.Marshal(modelsws.ErrorPayload{
			Code:    "get_project_failed",
			Message: err.Error(),
		})
		subID := sub.ID()
		return &eventsws.OutgoingEvent{
			To:             []eventsws.OutgoingTo{{ID: sub.ClientID()}},
			Type:           "error",
			Payload:        payload,
			SubscriptionID: &subID,
		}
	}

	return p.handler.buildProjectDetailEvent(sub, project)
}

// buildProjectDetailEvent builds an outgoing event with a single project for a subscription.
func (h *Handler) buildProjectDetailEvent(sub wssubscriptions.Subscription, project *domain.Project) *eventsws.OutgoingEvent {
	payload, err := json.Marshal(&modelsws.ProjectResponse{Project: *project})
	if err != nil {
		return nil
	}

	subID := sub.ID()
	return &eventsws.OutgoingEvent{
		To:             []eventsws.OutgoingTo{{ID: sub.ClientID()}},
		Type:           modelsws.MsgTypeProject,
		Payload:        payload,
		SubscriptionID: &subID,
	}
}

// buildProjectDeletedEvent builds an outgoing event indicating a project was deleted.
func (h *Handler) buildProjectDeletedEvent(sub wssubscriptions.Subscription, projectID domain.ProjectID) *eventsws.OutgoingEvent {
	payload, err := json.Marshal(&modelsws.ProjectDeletedResponse{ID: projectID.String()})
	if err != nil {
		return nil
	}

	subID := sub.ID()
	return &eventsws.OutgoingEvent{
		To:             []eventsws.OutgoingTo{{ID: sub.ClientID()}},
		Type:           modelsws.MsgTypeProjectDeleted,
		Payload:        payload,
		SubscriptionID: &subID,
	}
}

// projectMembersQuery is the query type for projects.members subscriptions.
type projectMembersQuery struct {
	ProjectID domain.ProjectID `json:"project_id"`
}

// projectsMembersSubItem implements SubscriptionItem for project members subscriptions.
type projectsMembersSubItem struct {
	handler *Handler
}

func (p *projectsMembersSubItem) CreateSubscription(client *wsproto.Client, query json.RawMessage) (wssubscriptions.Subscription, error) {
	var q projectMembersQuery
	if err := json.Unmarshal(query, &q); err != nil {
		return nil, err
	}

	// Validate the project ID
	if q.ProjectID == domain.ProjectID(uuid.Nil) {
		return nil, ErrProjectNotFound
	}

	return wssubscriptions.NewSubscription(client, "projects.members", &q), nil
}

func (p *projectsMembersSubItem) OnInitial(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
	query, ok := sub.Query().(*projectMembersQuery)
	if !ok {
		return nil
	}
	return p.handler.buildMembersListEvent(sub, query.ProjectID)
}

// buildMembersListEvent builds an outgoing event with the members list for a subscription.
func (h *Handler) buildMembersListEvent(sub wssubscriptions.Subscription, projectID domain.ProjectID) *eventsws.OutgoingEvent {
	members, err := h.service.ListMembers(context.Background(), sub.UserID(), projectID)
	if err != nil {
		payload, _ := json.Marshal(modelsws.ErrorPayload{
			Code:    "list_members_failed",
			Message: err.Error(),
		})
		subID := sub.ID()
		return &eventsws.OutgoingEvent{
			To:             []eventsws.OutgoingTo{{ID: sub.ClientID()}},
			Type:           "error",
			Payload:        payload,
			SubscriptionID: &subID,
		}
	}

	payload, err := json.Marshal(&modelsws.MembersListResponse{Members: members})
	if err != nil {
		return nil
	}

	subID := sub.ID()
	return &eventsws.OutgoingEvent{
		To:             []eventsws.OutgoingTo{{ID: sub.ClientID()}},
		Type:           modelsws.MsgTypeMembersList,
		Payload:        payload,
		SubscriptionID: &subID,
	}
}
