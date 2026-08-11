package httptransport

import (
	"github.com/yyeart/personal-contribution/backend/internal/logger"
	users_service "github.com/yyeart/personal-contribution/backend/internal/services/users"
)

type Server struct {
	users  *users_service.UsersService
	logger *logger.Logger
}

func NewServer(
	users *users_service.UsersService,
	logger *logger.Logger,
) *Server {
	return &Server{
		users:  users,
		logger: logger,
	}
}
