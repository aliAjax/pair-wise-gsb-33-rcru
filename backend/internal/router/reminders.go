package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/config"
	"github.com/gbplantwiki/gbplantwiki/internal/handler"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
)

func registerReminderRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.CareReminderHandler, limiter *middleware.RateLimiter) {
	reminders := v1.Group("/reminders", middleware.AuthRequired(cfg))
	reminders.GET("", h.List)
	reminders.GET("/calendar", h.ListByMonth)
	reminders.POST("", limiter.Limit(), h.Create)
	reminders.PUT("/:id", h.Update)
	reminders.PUT("/:id/status", h.UpdateStatus)
	reminders.GET("/:id/transfer-targets", h.TransferTargets)
	reminders.POST("/:id/transfer", h.Transfer)
	reminders.DELETE("/:id/cancel", h.Cancel)
	reminders.DELETE("/:id", h.Delete)
}
