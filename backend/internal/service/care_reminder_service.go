package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/gbplantwiki/gbplantwiki/internal/constants"
	"github.com/gbplantwiki/gbplantwiki/internal/dto"
	"github.com/gbplantwiki/gbplantwiki/internal/model"
	"github.com/gbplantwiki/gbplantwiki/internal/repository"
	"github.com/gbplantwiki/gbplantwiki/internal/util"
)

// CareReminderService implements care reminder state machine logic.
// The state machine (pending -> done / overdue / awaiting_confirm) is
// intentionally mirrored in frontend button visibility, log templates, error
// codes and formatters.
type CareReminderService struct {
	reminderRepo *repository.CareReminderRepository
	gardenRepo   *repository.UserGardenRepository
	plantRepo    *repository.PlantSpeciesRepository
	db           *gorm.DB
	logger       *slog.Logger
}

// NewCareReminderService creates a CareReminderService.
func NewCareReminderService(
	reminderRepo *repository.CareReminderRepository,
	gardenRepo *repository.UserGardenRepository,
	plantRepo *repository.PlantSpeciesRepository,
	db *gorm.DB,
	logger *slog.Logger,
) *CareReminderService {
	return &CareReminderService{
		reminderRepo: reminderRepo,
		gardenRepo:   gardenRepo,
		plantRepo:    plantRepo,
		db:           db,
		logger:       logger,
	}
}

// Create adds a reminder for the current user. A brand-new reminder starts
// its own series (SeriesID == ID after insert) at seq=1. At most one open
// occurrence per series can exist, enforced here and by CompleteOccurrence.
func (s *CareReminderService) Create(userID uint, m *model.CareReminder) (*model.CareReminder, error) {
	m.UserID = userID
	if m.Status == "" {
		m.Status = model.ReminderPending
	}
	if !isValidFrequency(m.Frequency) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareReminder[task_title=%s] create failed: frequency=%s unsupported",
				m.TaskTitle, m.Frequency))
	}
	if m.RemindDate.IsZero() {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareReminder[task_title=%s] create failed: remind_date required", m.TaskTitle))
	}
	if m.GardenID != 0 {
		garden, err := s.gardenRepo.FindOwned(nil, m.GardenID, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("CareReminder create failed: UserGarden[id=%d] not found or removed", m.GardenID))
			}
			return nil, fmt.Errorf("care reminder garden check: %w", err)
		}
		m.PlantSpeciesID = garden.PlantSpeciesID
	}
	m.Seq = 1
	m.ScheduleVersion = 1
	if txErr := s.db.Transaction(func(tx *gorm.DB) error {
		txReminder := s.reminderRepo.WithTx(tx)
		if err := txReminder.Create(m); err != nil {
			s.logger.Error(fmt.Sprintf(constants.LogReminderCreateFailed, m.TaskTitle), "error", err)
			return fmt.Errorf("care reminder create: %w", err)
		}
		// The first occurrence anchors its own series in the same tx so a
		// crashed second statement never leaves a series_id=0 row behind.
		m.SeriesID = m.ID
		if err := txReminder.Update(m); err != nil {
			return fmt.Errorf("care reminder series anchor: %w", err)
		}
		return nil
	}); txErr != nil {
		return nil, txErr
	}
	s.logger.Info(fmt.Sprintf(constants.LogReminderCreateSuccess, m.TaskTitle),
		"id", m.ID, "series_id", m.SeriesID)
	return m, nil
}

// ListByUser lists reminders with status filter, enriched with plant source.
func (s *CareReminderService) ListByUser(userID uint, status string) ([]dto.ReminderView, error) {
	if _, err := s.reminderRepo.MarkOverdue(userID); err != nil {
		s.logger.Warn("care reminder overdue mark failed", "error", err)
	}
	items, err := s.reminderRepo.ListByUser(userID, status)
	if err != nil {
		return nil, fmt.Errorf("care reminder list: %w", err)
	}
	return s.buildViews(items)
}

// ListByMonth lists reminders within a calendar month, enriched with plant source.
func (s *CareReminderService) ListByMonth(userID uint, year, month int) ([]dto.ReminderView, error) {
	items, err := s.reminderRepo.ListByMonth(userID, year, month)
	if err != nil {
		return nil, fmt.Errorf("care reminder month list: %w", err)
	}
	return s.buildViews(items)
}

// buildViews joins plant species and garden pot info onto reminder rows so the
// UI can show plant source and pot status.
func (s *CareReminderService) buildViews(items []model.CareReminder) ([]dto.ReminderView, error) {
	if len(items) == 0 {
		return []dto.ReminderView{}, nil
	}
	plantIDs := map[uint]struct{}{}
	gardenIDs := map[uint]struct{}{}
	for _, it := range items {
		if it.PlantSpeciesID != 0 {
			plantIDs[it.PlantSpeciesID] = struct{}{}
		}
		if it.GardenID != 0 {
			gardenIDs[it.GardenID] = struct{}{}
		}
	}
	plants, err := s.plantRepo.FindByIDs(keys(plantIDs))
	if err != nil {
		return nil, fmt.Errorf("care reminder plants: %w", err)
	}
	plantMap := map[uint]model.PlantSpecies{}
	for _, p := range plants {
		plantMap[p.ID] = p
	}
	gardens, err := s.gardenRepo.ListByIDs(keys(gardenIDs))
	if err != nil {
		return nil, fmt.Errorf("care reminder gardens: %w", err)
	}
	gardenMap := map[uint]model.UserGarden{}
	for _, g := range gardens {
		gardenMap[g.ID] = g
	}

	views := make([]dto.ReminderView, 0, len(items))
	for _, it := range items {
		v := dto.ReminderView{
			ID: it.ID, UserID: it.UserID, PlantSpeciesID: it.PlantSpeciesID,
			GardenID: it.GardenID, SeriesID: it.SeriesID, Seq: it.Seq,
			ScheduleVersion: it.ScheduleVersion, TaskTitle: it.TaskTitle,
			RemindDate: it.RemindDate, Frequency: it.Frequency, Status: it.Status,
			CreatedAt: it.CreatedAt,
		}
		if p, ok := plantMap[it.PlantSpeciesID]; ok {
			v.PlantName = p.Name
			v.PlantType = p.Type
			v.PlantAlias = p.Alias
		}
		if g, ok := gardenMap[it.GardenID]; ok {
			v.GardenName = g.Nickname
			v.Location = g.Location
			v.GardenStatus = g.Status
		}
		v.SourceLabel = buildReminderSource(v)
		views = append(views, v)
	}
	return views, nil
}

func buildReminderSource(v dto.ReminderView) string {
	switch {
	case v.GardenID != 0 && v.GardenName != "":
		return fmt.Sprintf("%s（%s）", v.PlantName, v.GardenName)
	case v.PlantName != "":
		return v.PlantName
	default:
		return "通用任务"
	}
}

func keys(m map[uint]struct{}) []uint {
	out := make([]uint, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Complete marks an open occurrence done and, for recurring tasks, generates
// exactly one next-period occurrence. It is idempotent: a repeated call
// (double window, double click) matches zero rows and returns processed=false
// with the already-generated next occurrence, instead of creating a second.
func (s *CareReminderService) Complete(userID, id uint) (*dto.ReminderCompleteResult, error) {
	result := &dto.ReminderCompleteResult{}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txReminder := s.reminderRepo.WithTx(tx)
		txGarden := s.gardenRepo.WithTx(tx)

		affected, err := txReminder.CompleteOpen(tx, id, userID)
		if err != nil {
			return fmt.Errorf("care reminder complete update: %w", err)
		}
		if affected == 0 {
			// Idempotent path: the row was already finished (or suspended).
			cur, ferr := txReminder.FindByID(id)
			if ferr != nil {
				if errors.Is(ferr, repository.ErrNotFound) {
					return util.NewAppError(404, constants.CodeNotFound,
						fmt.Sprintf("CareReminder[id=%d] not found", id))
				}
				return fmt.Errorf("care reminder complete refind: %w", ferr)
			}
			if cur.UserID != userID {
				return util.NewAppError(403, constants.CodeForbidden,
					fmt.Sprintf("CareReminder[id=%d] complete failed: user_id=%d not owner", id, userID))
			}
			if cur.Status == model.ReminderAwaitingConfirm {
				return util.NewAppError(409, constants.CodeConflict,
					fmt.Sprintf("CareReminder[id=%d] complete failed: awaiting confirmation after plant removal", id))
			}
			open, oerr := txReminder.FindOpenBySeries(tx, cur.SeriesID)
			var nextID uint
			if oerr == nil {
				nextID = open.ID
			} else if !errors.Is(oerr, repository.ErrNotFound) {
				return fmt.Errorf("care reminder complete next lookup: %w", oerr)
			}
			views, berr := s.buildViewsWithHandles([]model.CareReminder{*cur}, txReminder, txGarden)
			if berr != nil {
				return berr
			}
			result.Reminder = views[0]
			result.Processed = false
			result.NextReminderID = nextID
			result.Message = constants.MsgReminderAlreadyDone
			s.logger.Info(fmt.Sprintf(constants.LogReminderDuplicateComplete, id, cur.SeriesID), "user_id", userID)
			return nil
		}

		done, err := txReminder.FindByID(id)
		if err != nil {
			return fmt.Errorf("care reminder complete refind: %w", err)
		}

		// Recurring tasks occupy exactly one next-period slot: only generate a
		// successor when none exists for the series.
		nextID := uint(0)
		if done.Frequency != model.FrequencyNone {
			if _, err := txReminder.FindOpenBySeries(tx, done.SeriesID); errors.Is(err, repository.ErrNotFound) {
				nextDate := nextRemindDate(done.RemindDate, done.Frequency)
				if !nextDate.IsZero() {
					successor := &model.CareReminder{
						UserID:          done.UserID,
						PlantSpeciesID:  done.PlantSpeciesID,
						GardenID:        done.GardenID,
						SeriesID:        done.SeriesID,
						Seq:             done.Seq + 1,
						ScheduleVersion: done.ScheduleVersion,
						TaskTitle:       done.TaskTitle,
						RemindDate:      nextDate,
						Frequency:       done.Frequency,
						Status:          model.ReminderPending,
					}
					if err := txReminder.Create(successor); err != nil {
						return fmt.Errorf("care reminder successor create: %w", err)
					}
					nextID = successor.ID
					s.logger.Info(fmt.Sprintf(constants.LogReminderNextGenerated, successor.ID, done.SeriesID, successor.Seq),
						"remind_date", util.FormatDate(nextDate))
				}
			}
		}

		views, err := s.buildViewsWithHandles([]model.CareReminder{*done}, txReminder, txGarden)
		if err != nil {
			return err
		}
		result.Reminder = views[0]
		result.Processed = true
		result.NextReminderID = nextID
		result.Message = constants.MsgReminderDone
		s.logger.Info(fmt.Sprintf(constants.LogReminderStatusChanged, id, model.ReminderDone),
			"series_id", done.SeriesID, "next_id", nextID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// buildViewsWithHandles assembles views using transaction-bound repositories.
func (s *CareReminderService) buildViewsWithHandles(items []model.CareReminder,
	txReminder *repository.CareReminderRepository, txGarden *repository.UserGardenRepository,
) ([]dto.ReminderView, error) {
	plantIDs := map[uint]struct{}{}
	gardenIDs := map[uint]struct{}{}
	for _, it := range items {
		if it.PlantSpeciesID != 0 {
			plantIDs[it.PlantSpeciesID] = struct{}{}
		}
		if it.GardenID != 0 {
			gardenIDs[it.GardenID] = struct{}{}
		}
	}
	plants, err := s.plantRepo.FindByIDs(keys(plantIDs))
	if err != nil {
		return nil, fmt.Errorf("care reminder plants: %w", err)
	}
	plantMap := map[uint]model.PlantSpecies{}
	for _, p := range plants {
		plantMap[p.ID] = p
	}
	gardens, err := txGarden.ListByIDs(keys(gardenIDs))
	if err != nil {
		return nil, fmt.Errorf("care reminder gardens: %w", err)
	}
	gardenMap := map[uint]model.UserGarden{}
	for _, g := range gardens {
		gardenMap[g.ID] = g
	}
	views := make([]dto.ReminderView, 0, len(items))
	for _, it := range items {
		v := dto.ReminderView{
			ID: it.ID, UserID: it.UserID, PlantSpeciesID: it.PlantSpeciesID,
			GardenID: it.GardenID, SeriesID: it.SeriesID, Seq: it.Seq,
			ScheduleVersion: it.ScheduleVersion, TaskTitle: it.TaskTitle,
			RemindDate: it.RemindDate, Frequency: it.Frequency, Status: it.Status,
			CreatedAt: it.CreatedAt,
		}
		if p, ok := plantMap[it.PlantSpeciesID]; ok {
			v.PlantName = p.Name
			v.PlantType = p.Type
			v.PlantAlias = p.Alias
		}
		if g, ok := gardenMap[it.GardenID]; ok {
			v.GardenName = g.Nickname
			v.Location = g.Location
			v.GardenStatus = g.Status
		}
		v.SourceLabel = buildReminderSource(v)
		views = append(views, v)
	}
	return views, nil
}

// UpdateSchedule changes a reminder series. Editing the open occurrence
// directly replaces the slot; editing a finished occurrence bumps the
// schedule version, deletes the stale pre-generated next occurrence and
// rebuilds it from the new anchor. Either way, each series keeps exactly one
// open next-period slot.
func (s *CareReminderService) UpdateSchedule(userID, id uint, req dto.ReminderUpdateRequest) (*dto.ReminderCompleteResult, error) {
	if req.Frequency != nil && !isValidFrequency(*req.Frequency) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareReminder[id=%d] update failed: frequency=%s unsupported", id, *req.Frequency))
	}
	if req.RemindDate != nil && req.RemindDate.IsZero() {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareReminder[id=%d] update failed: remind_date required", id))
	}
	result := &dto.ReminderCompleteResult{}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txReminder := s.reminderRepo.WithTx(tx)
		txGarden := s.gardenRepo.WithTx(tx)

		target, err := txReminder.FindByID(id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("CareReminder[id=%d] not found", id))
			}
			return fmt.Errorf("care reminder update find: %w", err)
		}
		if target.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("CareReminder[id=%d] update failed: user_id=%d not owner", id, userID))
		}
		if target.Status == model.ReminderAwaitingConfirm {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("CareReminder[id=%d] update failed: awaiting confirmation after plant removal", id))
		}

		oldFrequency := target.Frequency
		if req.TaskTitle != nil {
			target.TaskTitle = *req.TaskTitle
		}
		if req.Frequency != nil {
			target.Frequency = *req.Frequency
		}
		scheduleChanged := req.RemindDate != nil || req.Frequency != nil

		if target.IsOpen() {
			// The open occurrence IS the next-period slot: rewrite it in place.
			if req.RemindDate != nil {
				target.RemindDate = normalizeDate(*req.RemindDate)
			}
			target.ScheduleVersion++
			target.Status = model.ReminderPending
			if err := txReminder.Update(target); err != nil {
				return fmt.Errorf("care reminder open update: %w", err)
			}
			// Defensive: no other open row may share the series.
			if _, err := txReminder.DeleteOpenSuccessor(tx, target.SeriesID, target.ID); err != nil {
				return fmt.Errorf("care reminder stale successor cleanup: %w", err)
			}
		} else {
			// Editing a finished occurrence changes the rule for the future.
			target.ScheduleVersion++
			if err := txReminder.Update(target); err != nil {
				return fmt.Errorf("care reminder done update: %w", err)
			}
			if scheduleChanged {
				if _, err := txReminder.DeleteOpenSuccessor(tx, target.SeriesID, target.ID); err != nil {
					return fmt.Errorf("care reminder successor invalidate: %w", err)
				}
				anchor := target.RemindDate
				if req.RemindDate != nil {
					anchor = normalizeDate(*req.RemindDate)
				}
				// Rebuild the next slot only when the series still repeats.
				if target.Frequency != model.FrequencyNone {
					open, ferr := txReminder.FindOpenBySeries(tx, target.SeriesID)
					if ferr != nil && !errors.Is(ferr, repository.ErrNotFound) {
						return fmt.Errorf("care reminder successor lookup: %w", ferr)
					}
					if ferr == nil && open != nil {
						// A successor survived; align it to the new schedule.
						open.RemindDate = nextRemindDate(anchor, target.Frequency)
						open.Frequency = target.Frequency
						open.TaskTitle = target.TaskTitle
						open.ScheduleVersion = target.ScheduleVersion
						open.Status = model.ReminderPending
						if uerr := txReminder.Update(open); uerr != nil {
							return fmt.Errorf("care reminder successor align: %w", uerr)
						}
						result.NextReminderID = open.ID
					} else {
						maxSeq, merr := txReminder.MaxSeqOfSeries(tx, target.SeriesID)
						if merr != nil {
							return fmt.Errorf("care reminder max seq: %w", merr)
						}
						nextDate := nextRemindDate(anchor, target.Frequency)
						if !nextDate.IsZero() {
							successor := &model.CareReminder{
								UserID:          target.UserID,
								PlantSpeciesID:  target.PlantSpeciesID,
								GardenID:        target.GardenID,
								SeriesID:        target.SeriesID,
								Seq:             maxSeq + 1,
								ScheduleVersion: target.ScheduleVersion,
								TaskTitle:       target.TaskTitle,
								RemindDate:      nextDate,
								Frequency:       target.Frequency,
								Status:          model.ReminderPending,
							}
							if err := txReminder.Create(successor); err != nil {
								return fmt.Errorf("care reminder successor rebuild: %w", err)
							}
							result.NextReminderID = successor.ID
							s.logger.Info(fmt.Sprintf(constants.LogReminderNextRebuilt, successor.ID, target.SeriesID),
								"remind_date", util.FormatDate(nextDate), "old_frequency", oldFrequency)
						}
					}
				}
			}
		}

		views, err := s.buildViewsWithHandles([]model.CareReminder{*target}, txReminder, txGarden)
		if err != nil {
			return err
		}
		result.Reminder = views[0]
		result.Processed = true
		result.Message = constants.MsgReminderUpdated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func normalizeDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// UpdateStatus transitions a reminder to pending/done. "done" delegates to
// the idempotent Complete flow so the status endpoint shares the same
// next-period guarantee.
func (s *CareReminderService) UpdateStatus(userID, id uint, status string) (*dto.ReminderCompleteResult, error) {
	switch status {
	case model.ReminderDone:
		return s.Complete(userID, id)
	case model.ReminderPending:
		m, err := s.reopen(userID, id)
		if err != nil {
			return nil, err
		}
		views, err := s.buildViewsWithHandles([]model.CareReminder{*m}, s.reminderRepo, s.gardenRepo)
		if err != nil {
			return nil, err
		}
		return &dto.ReminderCompleteResult{Reminder: views[0], Processed: true, Message: constants.MsgReminderReopened}, nil
	default:
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("CareReminder[id=%d] status=%s invalid transition", id, status))
	}
}

func (s *CareReminderService) reopen(userID, id uint) (*model.CareReminder, error) {
	m, err := s.reminderRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("CareReminder[id=%d] not found", id))
		}
		return nil, fmt.Errorf("care reminder status find: %w", err)
	}
	if m.UserID != userID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("CareReminder[id=%d] status change failed: user_id=%d not owner", id, userID))
	}
	if m.Status == model.ReminderAwaitingConfirm {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("CareReminder[id=%d] reopen failed: awaiting confirmation, cancel or transfer first", id))
	}
	if m.RemindDate.Before(time.Now()) {
		m.Status = model.ReminderOverdue
	} else {
		m.Status = model.ReminderPending
	}
	if err := s.reminderRepo.Update(m); err != nil {
		return nil, fmt.Errorf("care reminder status update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogReminderStatusChanged, id, m.Status), "id", id)
	return m, nil
}

// Delete removes a reminder owned by the user.
func (s *CareReminderService) Delete(userID, id uint) error {
	m, err := s.reminderRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("care reminder delete find: %w", err)
	}
	if m.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("CareReminder[id=%d] delete failed: not owner", id))
	}
	if err := s.reminderRepo.Delete(id); err != nil {
		return fmt.Errorf("care reminder delete: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogReminderDeleted, id), "series_id", m.SeriesID)
	return nil
}

// ListAwaiting returns removed pots with reminders awaiting confirmation plus
// the transferable same-species pots.
func (s *CareReminderService) ListAwaiting(userID uint) ([]dto.RemovedGardenView, error) {
	gardens, err := s.gardenRepo.ListRemoved(userID)
	if err != nil {
		return nil, fmt.Errorf("awaiting gardens list: %w", err)
	}
	plantIDs := map[uint]struct{}{}
	for _, g := range gardens {
		plantIDs[g.PlantSpeciesID] = struct{}{}
	}
	plants, err := s.plantRepo.FindByIDs(keys(plantIDs))
	if err != nil {
		return nil, fmt.Errorf("awaiting plants list: %w", err)
	}
	plantMap := map[uint]model.PlantSpecies{}
	for _, p := range plants {
		plantMap[p.ID] = p
	}

	out := make([]dto.RemovedGardenView, 0, len(gardens))
	for _, g := range gardens {
		reminders, err := s.reminderRepo.ListByGarden(g.ID, model.ReminderAwaitingConfirm)
		if err != nil {
			return nil, fmt.Errorf("awaiting reminders list: %w", err)
		}
		if len(reminders) == 0 {
			continue
		}
		targets, err := s.gardenRepo.ListTransferTargets(nil, userID, g.PlantSpeciesID, g.ID)
		if err != nil {
			return nil, fmt.Errorf("transfer targets list: %w", err)
		}
		views, err := s.buildViews(reminders)
		if err != nil {
			return nil, err
		}
		out = append(out, dto.RemovedGardenView{
			Garden:    s.gardenView(g, plantMap[g.PlantSpeciesID]),
			Reminders: views,
			Targets:   s.gardenViews(targets, plantMap),
		})
	}
	return out, nil
}

// CancelAwaiting cancels (deletes) all awaiting reminders of a removed pot.
func (s *CareReminderService) CancelAwaiting(userID, gardenID uint) (int64, error) {
	var n int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if active, ferr := s.gardenRepo.WithTx(tx).FindOwned(nil, gardenID, userID); ferr == nil && active != nil {
			// Pot is still active: awaiting reminders cannot belong to it.
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("UserGarden[id=%d] is active, nothing to confirm", gardenID))
		} else if ferr != nil && !errors.Is(ferr, repository.ErrNotFound) {
			return fmt.Errorf("cancel awaiting find: %w", ferr)
		}
		// Verify ownership of the removed row directly.
		removed, err := s.gardenRepo.WithTx(tx).FindByID(gardenID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("UserGarden[id=%d] not found", gardenID))
			}
			return err
		}
		if removed.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("UserGarden[id=%d] cancel failed: not owner", gardenID))
		}
		n, err = s.reminderRepo.WithTx(tx).CancelOpenByGarden(tx, gardenID, userID)
		if err != nil {
			return fmt.Errorf("cancel awaiting: %w", err)
		}
		s.logger.Info(fmt.Sprintf(constants.LogReminderAwaitingCanceled, gardenID, n), "user_id", userID)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// TransferAwaiting moves awaiting reminders of a removed pot onto another
// active pot of the same species. Garden removal and reminder suspension are
// committed together elsewhere, but here the whole transfer is transactional:
// on failure nothing is moved and the original relations stay intact.
func (s *CareReminderService) TransferAwaiting(userID, sourceGardenID, targetGardenID uint) (int64, error) {
	var n int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		txGarden := s.gardenRepo.WithTx(tx)
		source, err := txGarden.FindByID(sourceGardenID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("UserGarden[id=%d] not found", sourceGardenID))
			}
			return fmt.Errorf("transfer source find: %w", err)
		}
		if source.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("UserGarden[id=%d] transfer failed: not owner", sourceGardenID))
		}
		target, err := txGarden.FindOwned(nil, targetGardenID, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound,
					fmt.Sprintf("UserGarden[id=%d] transfer target not found or removed", targetGardenID))
			}
			return fmt.Errorf("transfer target find: %w", err)
		}
		if target.PlantSpeciesID != source.PlantSpeciesID {
			return util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("transfer failed: target plant_species_id=%d differs from source=%d",
					target.PlantSpeciesID, source.PlantSpeciesID))
		}

		n, err = s.reminderRepo.WithTx(tx).ReattachOpenByGarden(tx, sourceGardenID, targetGardenID, userID, source.PlantSpeciesID)
		if err != nil {
			return fmt.Errorf("transfer reminders: %w", err)
		}
		if n == 0 {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("UserGarden[id=%d] transfer failed: no awaiting reminders", sourceGardenID))
		}
		s.logger.Info(fmt.Sprintf(constants.LogReminderTransferred, sourceGardenID, targetGardenID, n),
			"user_id", userID, "plant_species_id", source.PlantSpeciesID)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *CareReminderService) gardenView(g model.UserGarden, p model.PlantSpecies) dto.GardenView {
	v := dto.GardenView{
		ID: g.ID, UserID: g.UserID, PlantSpeciesID: g.PlantSpeciesID,
		Nickname: g.Nickname, OwnedSince: g.OwnedSince, Location: g.Location,
		CareReminderID: g.CareReminderID, Status: g.Status, CreatedAt: g.CreatedAt,
		PlantName: p.Name, PlantType: p.Type, PlantAlias: p.Alias,
		Origin: p.Origin, Family: p.Family, Genus: p.Genus,
	}
	v.DisplayLabel = buildGardenDisplay(v)
	return v
}

func (s *CareReminderService) gardenViews(gs []model.UserGarden, plantMap map[uint]model.PlantSpecies) []dto.GardenView {
	out := make([]dto.GardenView, 0, len(gs))
	for _, g := range gs {
		out = append(out, s.gardenView(g, plantMap[g.PlantSpeciesID]))
	}
	return out
}

func buildGardenDisplay(v dto.GardenView) string {
	if v.Nickname != "" && v.Nickname != v.PlantName {
		return fmt.Sprintf("%s（%s）", v.Nickname, v.PlantName)
	}
	return v.PlantName
}
