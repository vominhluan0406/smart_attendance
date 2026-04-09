package wal

import (
	"encoding/json"
	"log"
	"time"

	"github.com/smart-attendance/attendance-service/internal/model"
	"github.com/smart-attendance/attendance-service/internal/repository"
)

// Processor is a cron job that reads pending WAL entries and syncs them to the database.
type Processor struct {
	writer         *Writer
	attendanceRepo *repository.AttendanceRepository
	logRepo        *repository.AttendanceLogRepository
	interval       time.Duration
	stop           chan struct{}
}

// walPayload is the subset of LogTimeInput stored in WAL for replay.
type walPayload struct {
	UserID    string   `json:"user_id"`
	BranchID  string   `json:"branch_id"`
	ShiftID   *string  `json:"shift_id,omitempty"`
	WorkDate  string   `json:"work_date"`
	Method    string   `json:"method"`
	IP        string   `json:"ip"`
	Lat       *float64 `json:"lat,omitempty"`
	Lng       *float64 `json:"lng,omitempty"`
	CheckInAt string   `json:"check_in_at"`
	Status    string   `json:"status"`
}

// NewProcessor creates a WAL processor that runs at the given interval.
func NewProcessor(
	writer *Writer,
	attendanceRepo *repository.AttendanceRepository,
	logRepo *repository.AttendanceLogRepository,
	interval time.Duration,
) *Processor {
	return &Processor{
		writer:         writer,
		attendanceRepo: attendanceRepo,
		logRepo:        logRepo,
		interval:       interval,
		stop:           make(chan struct{}),
	}
}

// Start begins the cron job in a background goroutine.
func (p *Processor) Start() {
	go func() {
		log.Printf("[wal][processor] started (interval=%v)", p.interval)
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		// Run once immediately on start
		p.processOnce()

		for {
			select {
			case <-ticker.C:
				p.processOnce()
			case <-p.stop:
				log.Printf("[wal][processor] stopped")
				return
			}
		}
	}()
}

// Stop gracefully stops the processor.
func (p *Processor) Stop() {
	close(p.stop)
}

// processOnce reads all pending WAL entries and attempts to sync them to DB.
func (p *Processor) processOnce() {
	pending, err := p.writer.ReadPending()
	if err != nil {
		log.Printf("[wal][processor] ERROR reading pending entries: %v", err)
		return
	}

	if len(pending) == 0 {
		return
	}

	log.Printf("[wal][processor] found %d pending entries, processing...", len(pending))

	synced := 0
	failed := 0

	for _, entry := range pending {
		if err := p.syncEntry(entry); err != nil {
			log.Printf("[wal][processor] ERROR syncing entry %s: %v", entry.ID, err)
			failed++
			// If DB is still down, stop retrying this batch
			break
		}

		if err := p.writer.MarkSynced(entry.ID); err != nil {
			log.Printf("[wal][processor] ERROR marking synced %s: %v", entry.ID, err)
		}
		synced++
	}

	if synced > 0 {
		log.Printf("[wal][processor] synced %d entries (%d failed)", synced, failed)
	}

	// Cleanup old WAL files (older than 7 days, all synced)
	p.writer.Cleanup(7 * 24 * time.Hour)
}

// syncEntry replays a single WAL entry into the database.
func (p *Processor) syncEntry(entry Entry) error {
	var payload walPayload
	if err := json.Unmarshal([]byte(entry.Payload), &payload); err != nil {
		log.Printf("[wal][processor] skip malformed payload for %s: %v", entry.ID, err)
		// Mark as synced to avoid infinite retry on bad data
		return nil
	}

	// Parse check-in time
	checkInAt, err := time.Parse(time.RFC3339, payload.CheckInAt)
	if err != nil {
		checkInAt = entry.Timestamp
	}

	// Check if attendance log already exists (idempotency by ID)
	existingLog, _ := p.logRepo.FindByID(entry.ID)
	if existingLog != nil {
		// Already synced (maybe marked failed previously)
		return nil
	}

	// Create attendance log
	attLog := &model.AttendanceLog{
		UserID:   payload.UserID,
		BranchID: payload.BranchID,
		ShiftID:  payload.ShiftID,
		WorkDate: payload.WorkDate,
		LoggedAt: checkInAt,
		Method:   payload.Method,
		IPAddress: payload.IP,
		Lat:      payload.Lat,
		Lng:      payload.Lng,
	}
	attLog.ID = entry.ID // Preserve original ID for idempotency

	if err := p.logRepo.Create(attLog); err != nil {
		return err // DB likely still down
	}

	// Update or create daily attendance summary
	existing, err := p.attendanceRepo.FindTodayByUserAndDate(payload.UserID, payload.WorkDate)
	if err != nil {
		// Create new attendance record
		status := model.AttendanceStatus(payload.Status)
		if status == "" {
			status = model.StatusOnTime
		}
		att := &model.Attendance{
			UserID:   payload.UserID,
			BranchID: payload.BranchID,
			ShiftID:  payload.ShiftID,
			WorkDate: payload.WorkDate,
			CheckInAt: &checkInAt,
			Status:   status,
			Method:   payload.Method,
			IPAddress: payload.IP,
			Lat:      payload.Lat,
			Lng:      payload.Lng,
			Note:     "WAL recovery",
		}
		return p.attendanceRepo.Create(att)
	}

	// Update existing — set earliest check-in, latest check-out
	if existing.CheckInAt == nil || checkInAt.Before(*existing.CheckInAt) {
		existing.CheckInAt = &checkInAt
	}
	if existing.CheckOutAt == nil || checkInAt.After(*existing.CheckOutAt) {
		existing.CheckOutAt = &checkInAt
	}
	existing.Note = appendNote(existing.Note, "WAL recovery")
	return p.attendanceRepo.Update(existing)
}

func appendNote(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + " | " + addition
}
