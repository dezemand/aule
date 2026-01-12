# Subscriptions

Subscriptions enable real-time UI updates. When data changes on the backend, all subscribed clients receive updates automatically.

## Why Subscriptions?

Traditional REST requires polling or manual refresh. Subscriptions provide:

1. **Instant updates** - Changes appear immediately across all clients
2. **Efficient** - No polling, only send when data changes
3. **Consistent** - All clients see the same state
4. **Simple frontend code** - React Query integration handles caching

## Available Subscriptions

| Topic | Query | Description |
|-------|-------|-------------|
| `projects.list` | none | All projects the user has access to |
| `projects.detail` | `{ project_id }` | Single project details |
| `projects.members` | `{ project_id }` | Members of a specific project |

## Frontend Usage

### useSubscription Hook

The `useSubscription` hook combines subscription management with React Query:

```typescript
import { useSubscription } from "@/services/subscriptions/use-subscription";
import { queryKeys } from "@/lib/query";
import type { ProjectResponse, MembersListResponse } from "@/model/ws";

function ProjectDetail({ projectId }: { projectId: string }) {
  // Subscribe to project updates
  const { data: projectData, isLoading } = useSubscription<ProjectResponse>({
    queryKey: queryKeys.projects.detail(projectId),
    topic: "projects.detail",
    query: { project_id: projectId },
  });

  // Subscribe to members updates
  const { data: membersData } = useSubscription<MembersListResponse>({
    queryKey: queryKeys.projects.members(projectId),
    topic: "projects.members",
    query: { project_id: projectId },
  });

  const project = projectData?.payload?.project;
  const members = membersData?.payload?.members ?? [];

  if (isLoading) return <Loading />;
  if (!project) return <NotFound />;

  return (
    <div>
      <h1>{project.name}</h1>
      <MembersList members={members} />
    </div>
  );
}
```

### How It Works

1. **Subscribe**: Component mounts, sends `subscription.subscribe.req`
2. **Initial data**: Backend sends current data immediately
3. **Cache**: Data stored in React Query cache
4. **Updates**: Backend pushes changes, cache updates automatically
5. **Unsubscribe**: Component unmounts, sends `subscription.unsubscribe.req`

### Options

```typescript
useSubscription<TResult>({
  queryKey: QueryKey,      // React Query cache key
  topic: string,           // Subscription topic
  query?: object,          // Topic-specific query parameters
  staleTime?: number,      // How long before refetch (default: 60s)
});
```

## Backend Implementation

### Subscription Service

Located at `internal/backend/wsproto/subscriptions/`:

```
subscriptions/
├── service.go   # Main service: Register, Subscribe, Notify
├── item.go      # SubscriptionItem interface
├── handler.go   # WS message handlers for subscribe/unsubscribe
└── store.go     # In-memory subscription storage
```

### Implementing a New Subscription

#### 1. Define Query Type

```go
// In your service handler file
type projectMembersQuery struct {
    ProjectID domain.ProjectID `json:"project_id"`
}
```

#### 2. Implement SubscriptionItem

```go
type projectsMembersSubItem struct {
    handler *Handler
}

func (p *projectsMembersSubItem) CreateSubscription(
    client *wsproto.Client,
    query json.RawMessage,
) (wssubscriptions.Subscription, error) {
    var q projectMembersQuery
    if err := json.Unmarshal(query, &q); err != nil {
        return nil, err
    }

    // Validate
    if q.ProjectID == domain.ProjectID(uuid.Nil) {
        return nil, ErrProjectNotFound
    }

    return wssubscriptions.NewSubscription(client, "projects.members", &q), nil
}

func (p *projectsMembersSubItem) OnInitial(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
    query := sub.Query().(*projectMembersQuery)
    return p.handler.buildMembersListEvent(sub, query.ProjectID)
}
```

#### 3. Register the Subscription

In your handler's constructor:

```go
func NewHandler(bus *event.Bus, service *Service, subscriptions *wssubscriptions.Service) *Handler {
    h := &Handler{...}
    
    // Register subscription types
    subscriptions.Register("projects.members", &projectsMembersSubItem{handler: h})
    
    return h
}
```

#### 4. Build Event Helper

```go
func (h *Handler) buildMembersListEvent(sub wssubscriptions.Subscription, projectID domain.ProjectID) *eventsws.OutgoingEvent {
    members, err := h.service.ListMembers(context.Background(), sub.UserID(), projectID)
    if err != nil {
        // Return error event
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

    payload, _ := json.Marshal(&modelsws.MembersListResponse{Members: members})
    subID := sub.ID()
    return &eventsws.OutgoingEvent{
        To:             []eventsws.OutgoingTo{{ID: sub.ClientID()}},
        Type:           modelsws.MsgTypeMembersList,
        Payload:        payload,
        SubscriptionID: &subID,
    }
}
```

#### 5. Bridge Domain Events

Connect domain events to subscription notifications:

```go
// In SetupEventHandlers()
wssubscriptions.SubscribeToBus(h.subscriptions, eventsprojects.TopicMemberAdded, "projects.members",
    func(ctx context.Context, e event.Event[eventsprojects.MemberAddedEvent]) (func(wssubscriptions.Subscription) bool, wssubscriptions.NotifyFunc) {
        return func(sub wssubscriptions.Subscription) bool {
                // Filter: only notify subscriptions for this project
                query, ok := sub.Query().(*projectMembersQuery)
                if !ok {
                    return false
                }
                return query.ProjectID == e.Payload().ProjectID
            }, func(sub wssubscriptions.Subscription) *eventsws.OutgoingEvent {
                // Build the update payload
                query := sub.Query().(*projectMembersQuery)
                return h.buildMembersListEvent(sub, query.ProjectID)
            }
    })
```

#### 6. Publish Domain Events

In your service methods, publish events when data changes:

```go
func (s *Service) AddMember(ctx context.Context, userID domain.UserID, projectID domain.ProjectID, memberUserID domain.UserID, role domain.ProjectMemberRole) error {
    // ... add member to database ...

    // Publish event to trigger subscription notifications
    event.Publish(s.bus, eventsprojects.TopicMemberAdded.Event(eventsprojects.MemberAddedEvent{
        ProjectID:    projectID,
        MemberUserID: memberUserID,
        Role:         role,
        AddedBy:      userID,
    }))

    return nil
}
```

## Data Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client A  │     │   Client B  │     │   Client C  │
│ (subscribed)│     │ (subscribed)│     │(not subscr.)│
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │                   │     AddMember()   │
       │                   │◄──────────────────│
       │                   │                   │
       │           ┌───────┴───────┐           │
       │           │    Service    │           │
       │           │  AddMember()  │           │
       │           └───────┬───────┘           │
       │                   │                   │
       │           ┌───────┴───────┐           │
       │           │  Publish      │           │
       │           │  MemberAdded  │           │
       │           └───────┬───────┘           │
       │                   │                   │
       │           ┌───────┴───────┐           │
       │           │ SubscribeTo   │           │
       │           │ Bus bridge    │           │
       │           └───────┬───────┘           │
       │                   │                   │
       │      ┌────────────┴────────────┐      │
       │      │  Filter: ProjectID match │     │
       │      └────────────┬────────────┘      │
       │                   │                   │
  ┌────┴────┐         ┌────┴────┐              │
  │ Updated │         │ Updated │              │
  │ Members │         │ Members │              │
  └─────────┘         └─────────┘              │
```

## Subscription Protocol

### Subscribe Request

```json
{
  "type": "subscription.subscribe.req",
  "id": "...",
  "payload": {
    "topic": "projects.members",
    "query": { "project_id": "..." },
    "initial": true
  }
}
```

### Subscribe Acknowledgment

```json
{
  "type": "subscription.subscribe.ack",
  "reply_to": "<request-id>",
  "payload": {
    "subscription_id": "550e8400-..."
  }
}
```

### Subscription Update

```json
{
  "type": "projects.members.list",
  "subscription_id": "550e8400-...",
  "payload": {
    "members": [...]
  }
}
```

### Unsubscribe Request

```json
{
  "type": "subscription.unsubscribe.req",
  "payload": {
    "subscription_id": "550e8400-..."
  }
}
```

## Best Practices

### Frontend

1. **Always use queryKeys** - Define keys in `src/lib/query.ts` for consistency
2. **Match query to data scope** - Use project ID in key when subscribing to project-specific data
3. **Handle loading states** - Show skeleton/spinner while `isLoading` is true
4. **Provide defaults** - Use `?? []` for array data to avoid null checks

### Backend

1. **Filter precisely** - Only notify subscriptions that need the update
2. **Keep payloads efficient** - Send full list, not diffs (simpler, more reliable)
3. **Publish events from service layer** - Not from handlers
4. **Use consistent event naming** - `Topic{Entity}{Action}` pattern
