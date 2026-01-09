package projectsservice

import "github.com/dezemandje/aule/internal/domain"

const (
	MsgTypeProjectsListReq = "projects.list.req"
	MsgTypeProjectsList    = "projects.list"
	MsgTypeProjectCreate   = "projects.create"
)

type ProjectsListRequest struct {
}

type ProjectsListResponse struct {
	Projects []domain.Project `json:"projects"`
}

func (p *ProjectsListRequest) Type() string {
	return MsgTypeProjectsListReq
}

func (p *ProjectsListResponse) Type() string {
	return MsgTypeProjectsList
}
