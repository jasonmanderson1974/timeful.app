package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"schej.it/server/db"
	"schej.it/server/errs"
	"schej.it/server/models"
	"schej.it/server/responses"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if userId is set
		session := sessions.Default(c)
		if session.Get("userId") == nil {
			// User id is not set, user is not signed in!
			c.JSON(http.StatusUnauthorized, responses.Error{Error: errs.NotSignedIn})
			c.Abort()
			return
		}

		// Check if user with user id exists
		user := db.GetUserById(session.Get("userId").(string))

		if user == nil {
			c.JSON(http.StatusUnauthorized, responses.Error{Error: errs.UserDoesNotExist})
			c.Abort()
			return
		}

		// Enforce the invite-only allowlist on every request, not just at
		// sign-in. If a member is struck from the roll, their existing session
		// stops working on the next request (cookie sessions can't be revoked
		// server-side otherwise). Fail-open while the allowlist is empty.
		if !db.IsAccessAllowed(user.Email) {
			session.Delete("userId")
			session.Save()
			c.JSON(http.StatusUnauthorized, responses.Error{Error: errs.NotInvited})
			c.Abort()
			return
		}

		c.Set("authUser", user)

		c.Next()
	}
}

// CanInviteRequired must be chained AFTER AuthRequired. It rejects signed-in
// users who cannot invite (guests), gating the /admin group. Individual
// handlers further restrict management actions to admins (CanManageUsers).
func CanInviteRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userInterface, exists := c.Get("authUser")
		if !exists {
			c.JSON(http.StatusUnauthorized, responses.Error{Error: errs.NotSignedIn})
			c.Abort()
			return
		}
		user := userInterface.(*models.User)

		if !user.EffectiveRole().CanInvite() {
			c.JSON(http.StatusForbidden, responses.Error{Error: errs.NotAuthorized})
			c.Abort()
			return
		}

		c.Next()
	}
}
