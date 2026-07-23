// The Chronicle: a members-only, read-only archive of past gatherings (C10).
// Entries are auto-captured by the gathering scheduler (services/reminders).
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sirtom/server/db"
	"sirtom/server/errs"
	"sirtom/server/middleware"
	"sirtom/server/responses"
)

func InitChronicle(router *gin.RouterGroup) {
	chronicle := router.Group("/chronicle")
	// Members only — AuthRequired enforces a signed-in, allowlisted member.
	chronicle.GET("", middleware.AuthRequired(), getChronicle)
}

// @Summary Lists past gatherings (The Chronicle)
// @Description Members-only, read-only archive of gatherings that have taken place, most recent first. Auto-captured when a confirmed gathering's time passes.
// @Tags chronicle
// @Produce json
// @Success 200 {array} models.ChronicleEntry
// @Router /chronicle [get]
func getChronicle(c *gin.Context) {
	entries, err := db.GetChronicleEntries(200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.Error{Error: errs.Internal})
		return
	}
	c.JSON(http.StatusOK, entries)
}
