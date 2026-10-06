package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/middleware"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// CareReminderHandler exposes care reminder endpoints.
type CareReminderHandler struct {
	svc    CareReminderServiceAPI
	logger *slog.Logger
}

// CareReminderServiceAPI is the handler-facing surface of the reminder service.
type CareReminderServiceAPI interface {
	Create(userID uint, m *model.CareReminder) (*model.CareReminder, error)
	ListByUser(userID uint, status string) ([]dto.ReminderView, error)
	ListByMonth(userID uint, year, month int) ([]dto.ReminderView, error)
	UpdateStatus(userID, id uint, status string) (*dto.ReminderCompleteResult, error)
	UpdateSchedule(userID, id uint, req dto.ReminderUpdateRequest) (*dto.ReminderCompleteResult, error)
	Complete(userID, id uint) (*dto.ReminderCompleteResult, error)
	Delete(userID, id uint) error
	ListAwaiting(userID uint) ([]dto.RemovedGardenView, error)
	CancelAwaiting(userID, gardenID uint) (int64, error)
	TransferAwaiting(userID, sourceGardenID, targetGardenID uint) (int64, error)
}

// NewCareReminderHandler creates a CareReminderHandler.
func NewCareReminderHandler(svc CareReminderServiceAPI, logger *slog.Logger) *CareReminderHandler {
	return &CareReminderHandler{svc: svc, logger: logger}
}

// List handles GET /reminders.
func (h *CareReminderHandler) List(c *gin.Context) {
	status := c.Query("status")
	items, err := h.svc.ListByUser(middleware.GetUserID(c), status)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// ListByMonth handles GET /reminders/calendar?year=&month=.
func (h *CareReminderHandler) ListByMonth(c *gin.Context) {
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
	items, err := h.svc.ListByMonth(middleware.GetUserID(c), year, month)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// Create handles POST /reminders.
func (h *CareReminderHandler) Create(c *gin.Context) {
	var req dto.ReminderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	m := &model.CareReminder{
		PlantSpeciesID: req.PlantSpeciesID, GardenID: req.GardenID,
		TaskTitle: req.TaskTitle, RemindDate: req.RemindDate, Frequency: req.Frequency,
	}
	created, err := h.svc.Create(middleware.GetUserID(c), m)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(created))
}

// Update handles PUT /reminders/:id — change date/frequency, invalidating the
// pre-generated next occurrence and recalculating it.
func (h *CareReminderHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid reminder id"))
		return
	}
	var req dto.ReminderUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	result, err := h.svc.UpdateSchedule(middleware.GetUserID(c), uint(id), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// Complete handles POST /reminders/:id/complete — idempotent completion that
// generates exactly one next-period occurrence per series.
func (h *CareReminderHandler) Complete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid reminder id"))
		return
	}
	result, err := h.svc.Complete(middleware.GetUserID(c), uint(id))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(result))
}

// UpdateStatus handles PUT /reminders/:id/status.
func (h *CareReminderHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid reminder id"))
		return
	}
	var req dto.ReminderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	m, err := h.svc.UpdateStatus(middleware.GetUserID(c), uint(id), req.Status)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(m))
}

// Awaiting handles GET /reminders/awaiting — removed pots' unconfirmed reminders.
func (h *CareReminderHandler) Awaiting(c *gin.Context) {
	items, err := h.svc.ListAwaiting(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(items))
}

// CancelAwaiting handles DELETE /gardens/:id/reminders/awaiting.
func (h *CareReminderHandler) CancelAwaiting(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid garden id"))
		return
	}
	n, err := h.svc.CancelAwaiting(middleware.GetUserID(c), uint(id))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"canceled": n}))
}

// TransferAwaiting handles POST /gardens/:id/reminders/transfer.
func (h *CareReminderHandler) TransferAwaiting(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid garden id"))
		return
	}
	var req dto.GardenTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	n, err := h.svc.TransferAwaiting(middleware.GetUserID(c), uint(id), req.TargetGardenID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"transferred": n}))
}

// Delete handles DELETE /reminders/:id.
func (h *CareReminderHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid reminder id"))
		return
	}
	if err := h.svc.Delete(middleware.GetUserID(c), uint(id)); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(gin.H{"deleted": true}))
}
