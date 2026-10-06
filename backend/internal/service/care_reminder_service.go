package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
	"gorm.io/gorm"
)

// CareReminderService implements care reminder state machine logic.
//
// The state machine (pending/overdue -> done; removing a pot moves open
// reminders to unbound; unbound -> pending via transfer or cancel via delete)
// is intentionally mirrored in frontend button visibility, log templates,
// error codes and formatters.
//
// Recurrence rule: every plan (series_key) occupies exactly one open
// next-occurrence slot. Completing it rolls the slot to the next date by
// frequency and is idempotent — a duplicate completion returns the already
// processed plan instead of creating a second next occurrence.
type CareReminderService struct {
	repo      *repository.CareReminderRepository
	gardenRepo *repository.UserGardenRepository
	plantRepo *repository.PlantSpeciesRepository
	logger    *slog.Logger
}

// NewCareReminderService creates a CareReminderService.
func NewCareReminderService(repo *repository.CareReminderRepository, gardenRepo *repository.UserGardenRepository, plantRepo *repository.PlantSpeciesRepository, logger *slog.Logger) *CareReminderService {
	return &CareReminderService{repo: repo, gardenRepo: gardenRepo, plantRepo: plantRepo, logger: logger}
}

// normalizeDate truncates to a calendar date in local time.
func normalizeDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// nextOccurrence advances a date by the reminder frequency.
func nextOccurrence(date time.Time, frequency string) time.Time {
	switch frequency {
	case constants.FrequencyDaily:
		return date.AddDate(0, 0, 1)
	case constants.FrequencyWeekly:
		return date.AddDate(0, 0, 7)
	case constants.FrequencyMonthly:
		return date.AddDate(0, 1, 0)
	case constants.FrequencyYearly:
		return date.AddDate(1, 0, 0)
	default:
		return time.Time{} // one-shot reminders have no next occurrence
	}
}

// seriesKey builds the plan identity: one slot per user + pot + task + frequency.
func seriesKey(userID, gardenID uint, taskTitle, frequency string) string {
	if frequency == "" {
		frequency = constants.FrequencyOnce
	}
	return fmt.Sprintf("u%d:g%d:%s:%s", userID, gardenID, frequency, taskTitle)
}

// Create adds a reminder for the current user. A plan may only hold one open
// next-occurrence slot per frequency.
func (s *CareReminderService) Create(userID uint, m *model.CareReminder) (*model.CareReminder, error) {
	m.UserID = userID
	if m.RemindDate.IsZero() {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareReminder[task_title=%s] create failed: remind_date required", m.TaskTitle))
	}
	m.RemindDate = normalizeDate(m.RemindDate)
	if m.Frequency == "" {
		m.Frequency = constants.FrequencyOnce
	}
	if !constants.IsValidReminderFrequency(m.Frequency) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareReminder[task_title=%s] create failed: frequency=%s invalid", m.TaskTitle, m.Frequency))
	}
	if m.Status == "" {
		m.Status = model.ReminderPending
	}
	if err := s.resolveSource(userID, m); err != nil {
		return nil, err
	}
	m.SeriesKey = seriesKey(userID, m.GardenID, m.TaskTitle, m.Frequency)
	m.ScheduleVersion = 1

	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		taken, err := s.repo.ExistsOpenAtDate(tx, userID, m.SeriesKey, m.RemindDate, 0)
		if err != nil {
			return fmt.Errorf("care reminder slot check: %w", err)
		}
		if taken {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareReminder[series=%s date=%s] create failed: %s", m.SeriesKey, util.FormatDate(m.RemindDate), constants.MsgReminderSlotTaken))
		}
		if err := s.repo.CreateTx(tx, m); err != nil {
			if errors.Is(err, repository.ErrDuplicate) {
				return util.NewAppError(409, constants.CodeConflict,
					fmt.Sprintf("CareReminder[series=%s date=%s] create failed: %s", m.SeriesKey, util.FormatDate(m.RemindDate), constants.MsgReminderSlotTaken))
			}
			return fmt.Errorf("care reminder create: %w", err)
		}
		return nil
	})
	if err != nil {
		var appErr *util.AppError
		if !errors.As(err, &appErr) {
			s.logger.Error(fmt.Sprintf(constants.LogReminderCreateFailed, m.TaskTitle), "error", err)
		}
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogReminderCreateSuccess, m.TaskTitle), "id", m.ID, "series", m.SeriesKey)
	return m, nil
}

// resolveSource validates garden/plant linkage and fills both ids on the model.
func (s *CareReminderService) resolveSource(userID uint, m *model.CareReminder) error {
	if m.GardenID != 0 {
		g, err := s.gardenRepo.FindByID(m.GardenID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("UserGarden[id=%d] not found for CareReminder", m.GardenID))
			}
			return fmt.Errorf("care reminder garden lookup: %w", err)
		}
		if g.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("CareReminder garden_id=%d bind failed: user_id=%d not owner", m.GardenID, userID))
		}
		m.PlantSpeciesID = g.PlantSpeciesID
		return nil
	}
	if m.PlantSpeciesID != 0 {
		if _, err := s.plantRepo.FindByID(m.PlantSpeciesID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("PlantSpecies[id=%d] not found for CareReminder", m.PlantSpeciesID))
			}
			return fmt.Errorf("care reminder plant lookup: %w", err)
		}
	}
	return nil
}

// ListByUser lists reminders with status filter.
func (s *CareReminderService) ListByUser(userID uint, status string) ([]model.CareReminderView, error) {
	if _, err := s.repo.MarkOverdue(userID); err != nil {
		s.logger.Warn("care reminder overdue mark failed", "error", err)
	}
	items, err := s.repo.ListByUser(userID, status)
	if err != nil {
		return nil, fmt.Errorf("care reminder list: %w", err)
	}
	return items, nil
}

// ListByMonth lists reminders within a calendar month.
func (s *CareReminderService) ListByMonth(userID uint, year, month int) ([]model.CareReminderView, error) {
	items, err := s.repo.ListByMonth(userID, year, month)
	if err != nil {
		return nil, fmt.Errorf("care reminder month list: %w", err)
	}
	return items, nil
}

// Update edits a reminder. A change of remind_date or frequency invalidates
// the old next occurrence and recomputes the schedule in place (version +1,
// the old series/date slot is released automatically because the row moves).
func (s *CareReminderService) Update(userID, id uint, req TaskUpdateInput) (*model.CareReminder, error) {
	var result *model.CareReminder
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		m, err := s.loadOwnedForUpdate(tx, userID, id)
		if err != nil {
			return err
		}
		if m.Status == model.ReminderUnbound {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareReminder[id=%d] update failed: unbound reminder must be canceled or transferred first", id))
		}
		old := *m
		if req.TaskTitle != "" {
			m.TaskTitle = req.TaskTitle
		}
		scheduleChanged := false
		if !req.RemindDate.IsZero() {
			newDate := normalizeDate(req.RemindDate)
			if !newDate.Equal(m.RemindDate) {
				m.RemindDate = newDate
				scheduleChanged = true
			}
		}
		if req.Frequency != "" {
			if !constants.IsValidReminderFrequency(req.Frequency) {
				return util.NewAppError(422, constants.CodeValidationError,
					fmt.Sprintf("CareReminder[id=%d] update failed: frequency=%s invalid", id, req.Frequency))
			}
			if req.Frequency != m.Frequency {
				m.Frequency = req.Frequency
				scheduleChanged = true
			}
		}
		m.SeriesKey = seriesKey(userID, m.GardenID, m.TaskTitle, m.Frequency)
		// A finished historical occurrence edited again becomes the open slot
		// of the new plan; open reminders are recomputed in place.
		if scheduleChanged || m.Status == model.ReminderDone {
			m.ScheduleVersion = old.ScheduleVersion + 1
			m.Status = model.ReminderPending
			if m.RemindDate.Before(normalizeDate(time.Now())) {
				m.Status = model.ReminderOverdue
			}
		}
		taken, err := s.repo.ExistsOpenAtDate(tx, userID, m.SeriesKey, m.RemindDate, m.ID)
		if err != nil {
			return fmt.Errorf("care reminder update slot check: %w", err)
		}
		if taken {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareReminder[series=%s date=%s] update failed: %s", m.SeriesKey, util.FormatDate(m.RemindDate), constants.MsgReminderSlotTaken))
		}
		if err := s.repo.UpdateTx(tx, m); err != nil {
			if errors.Is(err, repository.ErrDuplicate) {
				return util.NewAppError(409, constants.CodeConflict,
					fmt.Sprintf("CareReminder[series=%s date=%s] update failed: %s", m.SeriesKey, util.FormatDate(m.RemindDate), constants.MsgReminderSlotTaken))
			}
			return fmt.Errorf("care reminder update: %w", err)
		}
		result = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogReminderUpdated, result.ID, result.ScheduleVersion))
	return result, nil
}

// TaskUpdateInput carries editable reminder fields.
type TaskUpdateInput struct {
	TaskTitle  string
	RemindDate time.Time
	Frequency  string
}

// loadOwnedForUpdate locks a reminder row and verifies ownership.
func (s *CareReminderService) loadOwnedForUpdate(tx *gorm.DB, userID, id uint) (*model.CareReminder, error) {
	m, err := s.repo.FindByIDTx(tx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CareReminder[id=%d] not found", id))
		}
		return nil, fmt.Errorf("care reminder find: %w", err)
	}
	if m.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("CareReminder[id=%d] access failed: user_id=%d not owner", id, userID))
	}
	return m, nil
}

// Complete marks an open occurrence done and, for recurring reminders, creates
// exactly one next occurrence. Repeated completion of the same occurrence is
// idempotent: the second call returns the already processed plan.
func (s *CareReminderService) Complete(userID, id uint) (*model.CareReminder, *model.CareReminder, bool, error) {
	var completed, next *model.CareReminder
	alreadyProcessed := false
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		m, err := s.loadOwnedForUpdate(tx, userID, id)
		if err != nil {
			return err
		}
		if m.Status == model.ReminderDone {
			// 幂等：并发/重复完成直接返回该计划当前的下一期，不再生成。
			alreadyProcessed = true
			active, aErr := s.repo.ActiveBySeries(tx, userID, m.SeriesKey)
			if aErr == nil {
				next = active
			} else if errors.Is(aErr, repository.ErrNotFound) {
				next = nil
			} else {
				return aErr
			}
			completed = m
			return nil
		}
		if m.Status == model.ReminderUnbound {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareReminder[id=%d] complete failed: unbound reminder must be canceled or transferred first", id))
		}
		m.Status = model.ReminderDone
		if err := s.repo.UpdateTx(tx, m); err != nil {
			return fmt.Errorf("care reminder complete update: %w", err)
		}
		completed = m

		nextDate := nextOccurrence(m.RemindDate, m.Frequency)
		if !nextDate.IsZero() {
			// 该计划的下一期位置可能已被改期/其他操作占用；存在则直接复用，
			// 保证“每个频率只占一个下一期位置”。
			existing, fErr := s.repo.ActiveBySeries(tx, userID, m.SeriesKey)
			if fErr != nil && !errors.Is(fErr, repository.ErrNotFound) {
				return fmt.Errorf("care reminder next occurrence lookup: %w", fErr)
			}
			if existing != nil && !existing.RemindDate.Before(nextDate) {
				next = existing
				s.logger.Info(fmt.Sprintf(constants.LogReminderAdvanceConflict, m.SeriesKey, util.FormatDate(next.RemindDate)))
				return nil
			}
			n := &model.CareReminder{
				UserID:          m.UserID,
				PlantSpeciesID: m.PlantSpeciesID,
				GardenID:        m.GardenID,
				TaskTitle:       m.TaskTitle,
				RemindDate:      nextDate,
				Frequency:       m.Frequency,
				Status:          model.ReminderPending,
				SeriesKey:       m.SeriesKey,
				ScheduleVersion: m.ScheduleVersion,
			}
			// 唯一索引 uk_reminder_series_date 兜底并发：冲突时返回已存在的位置。
			if err := s.repo.CreateTx(tx, n); err != nil {
				if errors.Is(err, repository.ErrDuplicate) {
					s.logger.Warn(fmt.Sprintf(constants.LogReminderAdvanceConflict, m.SeriesKey, util.FormatDate(nextDate)))
					dup, dErr := s.repo.ActiveBySeries(tx, userID, m.SeriesKey)
					if dErr != nil && !errors.Is(dErr, repository.ErrNotFound) {
						return dErr
					}
					next = dup
					return nil
				}
				return fmt.Errorf("care reminder next occurrence create: %w", err)
			}
			next = n
			s.logger.Info(fmt.Sprintf(constants.LogReminderAdvanced, m.SeriesKey, util.FormatDate(nextDate)), "id", n.ID)
		}
		return nil
	})
	if err != nil {
		return nil, nil, false, err
	}
	if alreadyProcessed {
		s.logger.Info(fmt.Sprintf(constants.LogReminderCompleteDup, id, completed.Status))
	}
	return completed, next, alreadyProcessed, nil
}

// UpdateStatus transitions a reminder to a new status. "done" goes through the
// recurring Complete flow; other values flip the open status directly.
func (s *CareReminderService) UpdateStatus(userID, id uint, status string) (*model.CareReminder, bool, error) {
	switch status {
	case model.ReminderDone:
		done, next, dup, err := s.Complete(userID, id)
		if err != nil {
			return nil, false, err
		}
		// 周期提醒返回新的下一期位置；单次提醒返回刚完成的记录。
		if next != nil {
			return next, dup, nil
		}
		return done, dup, nil
	case model.ReminderPending, model.ReminderOverdue:
		var out *model.CareReminder
		err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
			m, err := s.loadOwnedForUpdate(tx, userID, id)
			if err != nil {
				return err
			}
			if status == model.ReminderOverdue || m.RemindDate.Before(normalizeDate(time.Now())) {
				m.Status = model.ReminderOverdue
			} else {
				m.Status = model.ReminderPending
			}
			if err := s.repo.UpdateTx(tx, m); err != nil {
				return fmt.Errorf("care reminder status update: %w", err)
			}
			out = m
			return nil
		})
		if err != nil {
			return nil, false, err
		}
		s.logger.Info(fmt.Sprintf(constants.LogReminderStatusChanged, id, out.Status), "id", id)
		return out, false, nil
	default:
		return nil, false, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareReminder[id=%d] status=%s invalid transition", id, status))
	}
}

// Transfer moves an unbound reminder onto another pot of the same plant
// species (转养). The target must not already hold the plan's open slot.
func (s *CareReminderService) Transfer(userID, id, targetGardenID uint) (*model.CareReminder, error) {
	var out *model.CareReminder
	err := s.repo.DB().Transaction(func(tx *gorm.DB) error {
		m, err := s.loadOwnedForUpdate(tx, userID, id)
		if err != nil {
			return err
		}
		if m.Status != model.ReminderUnbound {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareReminder[id=%d] transfer failed: status=%s is not unbound", id, m.Status))
		}
		target, err := s.gardenRepo.FindByIDTx(tx, targetGardenID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("UserGarden[id=%d] not found", targetGardenID))
			}
			return err
		}
		if target.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("CareReminder[id=%d] transfer failed: user_id=%d not owner of target garden", id, userID))
		}
		if target.PlantSpeciesID != m.PlantSpeciesID {
			return util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("CareReminder[id=%d] transfer failed: target species=%d != reminder species=%d",
					id, target.PlantSpeciesID, m.PlantSpeciesID))
		}
		newKey := seriesKey(userID, targetGardenID, m.TaskTitle, m.Frequency)
		taken, err := s.repo.ExistsOpenAtDate(tx, userID, newKey, m.RemindDate, 0)
		if err != nil {
			return err
		}
		if taken {
			s.logger.Warn(fmt.Sprintf(constants.LogReminderTransferConflict, targetGardenID, newKey))
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareReminder transfer to UserGarden[id=%d] failed: %s", targetGardenID, constants.MsgReminderSlotTaken))
		}
		from := m.GardenID
		m.GardenID = targetGardenID
		m.SeriesKey = newKey
		m.ScheduleVersion++
		m.Status = model.ReminderPending
		if err := s.repo.UpdateTx(tx, m); err != nil {
			if errors.Is(err, repository.ErrDuplicate) {
				return util.NewAppError(409, constants.CodeConflict,
					fmt.Sprintf("CareReminder transfer to UserGarden[id=%d] failed: %s", targetGardenID, constants.MsgReminderSlotTaken))
			}
			return fmt.Errorf("care reminder transfer update: %w", err)
		}
		out = m
		s.logger.Info(fmt.Sprintf(constants.LogReminderTransfer, id, from, targetGardenID))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TransferTargets lists same-species pots a reminder can be transferred to.
func (s *CareReminderService) TransferTargets(userID, id uint) ([]model.UserGarden, error) {
	m, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("CareReminder[id=%d] not found", id))
		}
		return nil, err
	}
	if m.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("CareReminder[id=%d] transfer targets failed: not owner", id))
	}
	return s.gardenRepo.ListSameSpecies(s.gardenRepo.DB(), userID, m.PlantSpeciesID, m.GardenID)
}

// Cancel deletes an unbound reminder (取消待确认提醒).
func (s *CareReminderService) Cancel(userID, id uint) error {
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		m, err := s.loadOwnedForUpdate(tx, userID, id)
		if err != nil {
			return err
		}
		if m.Status != model.ReminderUnbound {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareReminder[id=%d] cancel failed: status=%s is not unbound", id, m.Status))
		}
		if err := s.repo.DeleteTx(tx, id); err != nil {
			return fmt.Errorf("care reminder cancel: %w", err)
		}
		return nil
	})
}

// Delete removes a reminder owned by the user.
func (s *CareReminderService) Delete(userID, id uint) error {
	m, err := s.repo.FindByID(id)
	if err != nil {
		return fmt.Errorf("care reminder delete find: %w", err)
	}
	if m.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden, fmt.Sprintf("CareReminder[id=%d] delete failed: not owner", id))
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("care reminder delete: %w", err)
	}
	return nil
}
