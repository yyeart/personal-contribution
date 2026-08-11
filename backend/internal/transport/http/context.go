package httptransport

import (
	"github.com/gin-gonic/gin"

	"github.com/yyeart/personal-contribution/backend/internal/domain/models"
)

const sessionContextKey = "session"

// CurrentUser достаёт пользователя из сессии, проверенной SessionMiddleware.
func CurrentUser(c *gin.Context) (models.UserID, bool) {
	value, exists := c.Get(sessionContextKey)
	if !exists {
		return models.UserID{}, false
	}

	session, ok := value.(models.Session)
	if !ok {
		return models.UserID{}, false
	}

	userID, err := models.NewUserID(session.UserID)
	if err != nil {
		return models.UserID{}, false
	}
	return userID, true
}

// CurrentSession достает текущую сессию из контекста.
func CurrentSession(c *gin.Context) (models.Session, bool) {
	value, exists := c.Get(sessionContextKey)
	if !exists {
		return models.Session{}, false
	}

	session, ok := value.(models.Session)
	if !ok {
		return models.Session{}, false
	}

	return session, true
}
