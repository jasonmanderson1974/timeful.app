/* The /admin group contains routes for managing the invite-only allowlist.
   All routes require a signed-in user with the canInvite role. */
package routes

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"schej.it/server/db"
	"schej.it/server/errs"
	"schej.it/server/logger"
	"schej.it/server/middleware"
	"schej.it/server/models"
	"schej.it/server/responses"
)

// isValidEmail does a basic RFC 5322 address check and rejects '+' aliases
// (mirrors the frontend's sign-in validation).
func isValidEmail(email string) bool {
	if email == "" || strings.Contains(email, "+") {
		return false
	}
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func InitAdmin(router *gin.RouterGroup) {
	adminRouter := router.Group("/admin")
	adminRouter.Use(middleware.AuthRequired(), middleware.CanInviteRequired())

	adminRouter.GET("/allowlist", getAllowlist)
	adminRouter.POST("/allowlist", addAllowlistEmail)
	adminRouter.DELETE("/allowlist", removeAllowlistEmail)
	adminRouter.POST("/member/can-invite", setMemberCanInvite)
}

// allowlistMember is an allowlist entry enriched with the registered-user
// status for that email, so the admin UI can show who has actually signed up.
type allowlistMember struct {
	models.AllowlistEntry
	HasAccount bool   `json:"hasAccount"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
	CanInvite  bool   `json:"canInvite"`
}

func authUserFromContext(c *gin.Context) *models.User {
	userInterface, _ := c.Get("authUser")
	return userInterface.(*models.User)
}

// @Summary Lists the invite-only allowlist (with member/account status)
// @Tags admin
// @Produce json
// @Success 200 {array} allowlistMember
// @Router /admin/allowlist [get]
func getAllowlist(c *gin.Context) {
	entries := db.GetAllowlist()
	members := make([]allowlistMember, 0, len(entries))
	for _, entry := range entries {
		m := allowlistMember{AllowlistEntry: entry}
		if user := db.GetUserByEmail(entry.Email); user != nil {
			m.HasAccount = true
			m.FirstName = user.FirstName
			m.LastName = user.LastName
			m.CanInvite = user.CanInvite != nil && *user.CanInvite
		}
		members = append(members, m)
	}
	c.JSON(http.StatusOK, members)
}

// @Summary Adds an email to the allowlist (invite a member)
// @Tags admin
// @Accept json
// @Produce json
// @Param payload body object{email=string} true "Email to invite"
// @Success 200
// @Router /admin/allowlist [post]
func addAllowlistEmail(c *gin.Context) {
	payload := struct {
		Email string `json:"email" binding:"required"`
	}{}
	if err := c.BindJSON(&payload); err != nil {
		return
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if !isValidEmail(email) {
		c.JSON(http.StatusBadRequest, responses.Error{Error: errs.InvalidEmail})
		return
	}

	admin := authUserFromContext(c)
	if err := db.AddToAllowlist(email, admin.Email); err != nil {
		logger.StdErr.Println(err)
		c.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"email": email})
}

// @Summary Removes an email from the allowlist
// @Tags admin
// @Accept json
// @Produce json
// @Param payload body object{email=string} true "Email to remove"
// @Success 200
// @Router /admin/allowlist [delete]
func removeAllowlistEmail(c *gin.Context) {
	payload := struct {
		Email string `json:"email" binding:"required"`
	}{}
	if err := c.BindJSON(&payload); err != nil {
		return
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))

	// Guard: don't let an admin remove their own access and lock themselves out.
	admin := authUserFromContext(c)
	if email == strings.ToLower(strings.TrimSpace(admin.Email)) {
		c.JSON(http.StatusBadRequest, responses.Error{Error: errs.CannotRemoveSelf})
		return
	}

	if err := db.RemoveFromAllowlist(email); err != nil {
		logger.StdErr.Println(err)
		c.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"email": email})
}

// @Summary Grants or revokes the canInvite role for a member
// @Tags admin
// @Accept json
// @Produce json
// @Param payload body object{email=string,canInvite=bool} true "Member email and desired canInvite state"
// @Success 200
// @Router /admin/member/can-invite [post]
func setMemberCanInvite(c *gin.Context) {
	payload := struct {
		Email     string `json:"email" binding:"required"`
		CanInvite *bool  `json:"canInvite" binding:"required"`
	}{}
	if err := c.BindJSON(&payload); err != nil {
		return
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))

	// Guard: an admin can't revoke their own inviter role (avoid self-lockout).
	admin := authUserFromContext(c)
	if email == strings.ToLower(strings.TrimSpace(admin.Email)) && !*payload.CanInvite {
		c.JSON(http.StatusBadRequest, responses.Error{Error: errs.CannotRemoveSelf})
		return
	}

	matched, err := db.SetUserCanInvite(email, *payload.CanInvite)
	if err != nil {
		logger.StdErr.Println(err)
		c.JSON(http.StatusInternalServerError, responses.Error{Error: err.Error()})
		return
	}
	if matched == 0 {
		// No account for that email yet — the role only applies once they sign in.
		c.JSON(http.StatusNotFound, responses.Error{Error: errs.UserDoesNotExist})
		return
	}

	c.JSON(http.StatusOK, gin.H{"email": email, "canInvite": *payload.CanInvite})
}
