// Package eventsprojects defines project domain events for the event bus.
package eventsprojects

import (
	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/event"
)

// Topics for project domain events.
var (
	TopicProjectCreated = event.NewTopic[ProjectCreatedEvent]("projects.created")
	TopicProjectUpdated = event.NewTopic[ProjectUpdatedEvent]("projects.updated")
	TopicProjectDeleted = event.NewTopic[ProjectDeletedEvent]("projects.deleted")

	// Member events
	TopicMemberAdded   = event.NewTopic[MemberAddedEvent]("projects.members.added")
	TopicMemberUpdated = event.NewTopic[MemberUpdatedEvent]("projects.members.updated")
	TopicMemberRemoved = event.NewTopic[MemberRemovedEvent]("projects.members.removed")
)

// ProjectCreatedEvent is published when a new project is created.
type ProjectCreatedEvent struct {
	ProjectID domain.ProjectID
	CreatorID domain.UserID
	Project   domain.Project
}

// ProjectUpdatedEvent is published when a project is updated.
type ProjectUpdatedEvent struct {
	ProjectID domain.ProjectID
	UpdaterID domain.UserID
	Project   domain.Project
}

// ProjectDeletedEvent is published when a project is deleted.
type ProjectDeletedEvent struct {
	ProjectID domain.ProjectID
	DeleterID domain.UserID
}

// MemberAddedEvent is published when a member is added to a project.
type MemberAddedEvent struct {
	ProjectID    domain.ProjectID
	MemberUserID domain.UserID
	Role         domain.ProjectMemberRole
	AddedBy      domain.UserID
}

// MemberUpdatedEvent is published when a member's role is updated.
type MemberUpdatedEvent struct {
	ProjectID    domain.ProjectID
	MemberUserID domain.UserID
	Role         domain.ProjectMemberRole
	UpdatedBy    domain.UserID
}

// MemberRemovedEvent is published when a member is removed from a project.
type MemberRemovedEvent struct {
	ProjectID    domain.ProjectID
	MemberUserID domain.UserID
	RemovedBy    domain.UserID
}
