# WebSocket Architecture

This document explains how WebSocket communication works between the frontend and backend.

## Overview

The WebSocket connection is the primary communication channel for the UI. All user interactions (except OAuth) flow through a single persistent WebSocket connection.

```
┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│    Frontend     │◄───────►│   WS Handler    │◄───────►│   Event Bus     │
│  WebSocketClient│   WS    │  (wsproto)      │  Events │                 │
└─────────────────┘         └─────────────────┘         └─────────────────┘
                                                                │
                                                    ┌───────────┴───────────┐
                                                    ▼                       ▼
                                            ┌─────────────┐         ┌─────────────┐
                                            │  Service    │         │  Service    │
                                            │  Handlers   │         │  Handlers   │
                                            └─────────────┘         └─────────────┘
```

## Message Envelope

Every WebSocket message uses the same envelope format:

```json
{
  "type": "projects.create.req",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "reply_to": null,
  "subscription_id": null,
  "seq": 42,
  "time": "2024-01-15T10:30:00Z",
  "payload": {
    "name": "My Project",
    "description": "A new project"
  }
}
```

| Field | Description |
|-------|-------------|
| `type` | Message type identifier (e.g., `projects.create.req`) |
| `id` | Unique UUID for this message |
| `reply_to` | UUID of the request this is responding to |
| `subscription_id` | UUID linking message to a subscription |
| `seq` | Sequence number for ordering |
| `time` | ISO 8601 timestamp |
| `payload` | Type-specific message content |

## Backend Architecture

### Key Components

```
internal/backend/wsproto/
├── handler.go   # WebSocket connection lifecycle, event publishing
├── client.go    # Client connection wrapper with thread-safe send
└── store.go     # In-memory client connection store
```

### Connection Lifecycle

1. **Upgrade**: HTTP request upgrades to WebSocket at `/api/ws?token=<jwt>`
2. **Authentication**: JWT token validated, user context established
3. **Client Creation**: `Client` struct created with unique ID
4. **Event Publishing**: `TopicConnect` event published to bus
5. **Message Loop**: Read messages, parse envelope, publish to event bus
6. **Disconnection**: `TopicDisconnect` event published, cleanup

### Event Bus Integration

The WebSocket handler doesn't process messages directly. Instead, it publishes events:

```go
// handler.go - All incoming messages become events
h.publishMessage(client, &envelope)  // Publishes to eventsws.TopicIncoming
```

Service handlers subscribe to these events:

```go
// In a service handler's SetupEventHandlers()
wsproto.WsToEvent(h.bus, modelsws.MsgTypeProjectCreate, topicCreateProject,
    func(payload modelsws.ProjectCreateRequest, evt event.Event[eventsws.IncomingEvent]) createProjectEvent {
        return createProjectEvent{
            ClientID:     evt.Payload().ClientID,
            UserID:       evt.Payload().UserID,
            Name:         payload.Name,
            Description:  payload.Description,
        }
    })
```

### Sending Responses

Responses go through the event bus too:

```go
// Publish outgoing event
event.Publish(h.bus, eventsws.TopicOutgoing.Event(eventsws.OutgoingEvent{
    To:      []eventsws.OutgoingTo{{ID: clientID}},
    Type:    modelsws.MsgTypeProjectCreated,
    Payload: payloadBytes,
    ReplyTo: &requestMsgID,
}))
```

The WS handler subscribes to `TopicOutgoing` and sends to clients:

```go
// handler.go - Outgoing events become WS messages
event.Subscribe(bus, eventsws.TopicOutgoing, h.handleOutgoingEvent)
```

## Frontend Architecture

### WebSocketClient

Located at `frontend/src/services/websocket/websocket-client.ts`:

```typescript
const client = new WebSocketClient({
  getToken: async () => authStore.getAccessToken(),
  initialRetryDelay: 1000,
  maxRetryDelay: 30000,
});

await client.connect();
```

Features:
- **Auto-reconnect**: Exponential backoff on disconnection
- **Token refresh**: Gets fresh token on each reconnect attempt
- **Message handlers**: Register callbacks for incoming messages
- **State tracking**: `disconnected` | `connecting` | `connected` | `reconnecting`

### Sending Messages

```typescript
// Send and get typed response
const response = await wsClient
  .send("projects.create.req", { name: "My Project" })
  .responseTypes({
    "projects.created": projectCreatedSchema,
    "error": errorPayloadSchema,
  });

if (response.type === "error") {
  console.error(response.payload.message);
} else {
  console.log("Created:", response.payload.project);
}
```

### React Integration

The `useWebSocket` hook provides access to the client:

```typescript
import { useWebSocket, useConnectionState } from "@/services/websocket/client";

function MyComponent() {
  const wsClient = useWebSocket();
  const connectionState = useConnectionState();
  
  if (connectionState !== "connected") {
    return <Loading />;
  }
  
  // Use wsClient.send(...)
}
```

## Message Flow Example

### Request: Create Project

```
Frontend                    Backend
   │                           │
   │ send("projects.create.req", {...})
   │─────────────────────────►│
   │                           │ Parse envelope
   │                           │ Publish to TopicIncoming
   │                           │─────►┌─────────────────┐
   │                           │      │ WsToEvent       │
   │                           │      │ transformer     │
   │                           │◄─────└─────────────────┘
   │                           │ Publish createProjectEvent
   │                           │─────►┌─────────────────┐
   │                           │      │ handleCreate    │
   │                           │      │ Project()       │
   │                           │◄─────└─────────────────┘
   │                           │ service.CreateProject()
   │                           │ sendResponse() → TopicOutgoing
   │ "projects.created"        │
   │◄─────────────────────────│
   │                           │
```

### Real-time Update: Subscription Notification

```
Frontend                    Backend
   │                           │
   │ (subscribed to "projects.list")
   │                           │
   │                           │ Another client creates project
   │                           │ service.CreateProject()
   │                           │ Publish ProjectCreatedEvent
   │                           │─────►┌─────────────────┐
   │                           │      │ SubscribeToBus  │
   │                           │      │ bridge          │
   │                           │◄─────└─────────────────┘
   │                           │ Notify matching subscriptions
   │                           │ buildProjectsListEvent()
   │ "projects.list"           │
   │◄─────────────────────────│ (fresh list with new project)
   │                           │
```

## Error Handling

### Backend Errors

Errors are sent as `error` type messages:

```go
h.sendError(clientID, replyTo, "create_project_failed", err.Error())

// Produces:
{
  "type": "error",
  "reply_to": "<original-request-id>",
  "payload": {
    "code": "create_project_failed",
    "message": "project name already exists"
  }
}
```

### Frontend Error Handling

```typescript
const response = await wsClient
  .send("projects.create.req", data)
  .responseTypes({
    "projects.created": projectCreatedSchema,
    "error": errorPayloadSchema,
  });

if (response.type === "error") {
  // Handle error
  toast.error(response.payload.message);
}
```

## Authentication

### Token-Based Auth

1. JWT token passed as query parameter: `/api/ws?token=<jwt>`
2. Backend validates token before upgrade
3. Connection lifetime bounded by token expiry
4. On token expiry, server sends `connection.close` message
5. Client reconnects with fresh token

### Auth Expiry Flow

```go
// handler.go
ctx, cancel := context.WithDeadlineCause(ctx, userToken.Expires(), ErrAuthExpired)

// When deadline hits:
env, _ := modelsws.NewEnvelope("connection.close", nil)
client.Send(env)
```

Frontend handles this:
```typescript
if (envelope.type === "connection.close") {
  this.ws?.close(1000, "Server requested disconnect");
  // Auto-reconnect will get fresh token
}
```
