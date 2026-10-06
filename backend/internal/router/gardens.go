package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/handler"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
)

func registerGardenRoutes(v1 *gin.RouterGroup, cfg *config.Config, gh *handler.UserGardenHandler, rh *handler.CareReminderHandler, limiter *middleware.RateLimiter) {
	gardens := v1.Group("/gardens", middleware.AuthRequired(cfg))
	gardens.GET("", gh.List)
	gardens.POST("", limiter.Limit(), gh.Add)
	gardens.PUT("/:id/reminder", gh.BindReminder)
	gardens.DELETE("/:id", gh.Remove)
	// After a plant is moved out, its unfinished reminders sit at
	// awaiting_confirm: the user cancels them or transfers them to another
	// pot of the same plant species.
	gardens.DELETE("/:id/reminders/awaiting", rh.CancelAwaiting)
	gardens.POST("/:id/reminders/transfer", limiter.Limit(), rh.TransferAwaiting)
}
