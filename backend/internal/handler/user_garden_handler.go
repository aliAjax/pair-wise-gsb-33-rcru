package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// UserGardenHandler exposes "my garden" endpoints.
type UserGardenHandler struct {
	svc    UserGardenServiceAPI
	logger *slog.Logger
}

// UserGardenServiceAPI is the handler-facing surface of the garden service.
type UserGardenServiceAPI interface {
	Add(userID uint, g *model.UserGarden) (*dto.GardenView, error)
	List(userID uint) ([]dto.GardenView, error)
	Remove(userID, id uint) (int64, error)
	BindReminder(userID, gardenID, reminderID uint) (*dto.GardenView, error)
}

// NewUserGardenHandler creates a UserGardenHandler.
func NewUserGardenHandler(svc UserGardenServiceAPI, logger *slog.Logger) *UserGardenHandler {
	return &UserGardenHandler{svc: svc, logger: logger}
}

// List handles GET /gardens.
func (h *UserGardenHandler) List(c *gin.Context) {
	items, err := h.svc.List(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// Add handles POST /gardens.
func (h *UserGardenHandler) Add(c *gin.Context) {
	var req dto.GardenAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	g := &model.UserGarden{
		PlantSpeciesID: req.PlantSpeciesID, Nickname: req.Nickname,
		OwnedSince: req.OwnedSince, Location: req.Location,
	}
	created, err := h.svc.Add(middleware.GetUserID(c), g)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(created))
}

// BindReminder handles PUT /gardens/:id/reminder.
func (h *UserGardenHandler) BindReminder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid garden id"))
		return
	}
	var req dto.GardenBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	g, err := h.svc.BindReminder(middleware.GetUserID(c), uint(id), req.ReminderID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(g))
}

// Remove handles DELETE /gardens/:id. Open reminders are suspended to
// awaiting_confirm in the same transaction; the response reports how many.
func (h *UserGardenHandler) Remove(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid garden id"))
		return
	}
	suspended, err := h.svc.Remove(middleware.GetUserID(c), uint(id))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"removed": true, "awaiting_reminders": suspended}))
}
