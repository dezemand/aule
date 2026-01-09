package api

import (
	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/database"
	"github.com/dezemandje/aule/internal/repository/postgres"
	projectsservice "github.com/dezemandje/aule/internal/service/project"
	userservice "github.com/dezemandje/aule/internal/service/user"
)

type Data struct {
	DB                     *database.DB
	RefreshTokenRepository auth.RefreshTokenRepository
	UserRepository         userservice.Repository
	ProjectRepository      projectsservice.Repository
}

func setupData(ctx *ApiContext) (err error) {
	ctx.Data = &Data{}

	ctx.Data.DB, err = database.New(&ctx.Config.DB)
	if err != nil {
		return err
	}

	ctx.Data.RefreshTokenRepository = postgres.NewRefreshTokenRepository(ctx.Data.DB)
	ctx.Data.UserRepository = postgres.NewUserRepository(ctx.Data.DB)
	ctx.Data.ProjectRepository = postgres.NewProjectRepository(ctx.Data.DB)

	return nil
}
