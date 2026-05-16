package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
	"github.com/xe-pc23/shift-notifier/internal/scheduler"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ShiftRecord struct {
	ID          uint `gorm:"primaryKey"`
	StaffName   string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
	SourceKey   string `gorm:"uniqueIndex;not null"`
	ContentHash string `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ShiftNotificationRecord struct {
	ID               string `gorm:"primaryKey"`
	ShiftID          uint   `gorm:"index;not null"`
	NotificationType string `gorm:"index;not null"`
	ScheduledFor     time.Time
	SentAt           *time.Time
	Status           string `gorm:"index;not null"`
	ErrorMessage     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ShiftImportRecord struct {
	ID           uint `gorm:"primaryKey"`
	SourceType   string
	FileName     string
	FilePath     string
	FileHash     string `gorm:"index;not null"`
	ImportedAt   *time.Time
	Status       string `gorm:"index;not null"`
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Store struct {
	db *gorm.DB
}

func OpenSQLite(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.New(stdlog.New(os.Stdout, "\r\n", stdlog.LstdFlags), logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		}),
	})
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&ShiftRecord{}, &ShiftNotificationRecord{}, &ShiftImportRecord{})
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertShifts(shifts []model.Shift) ([]model.Shift, error) {
	var saved []model.Shift

	err := s.db.Transaction(func(tx *gorm.DB) error {
		store := &Store{db: tx}
		var err error
		saved, err = store.upsertShifts(shifts)
		return err
	})
	if err != nil {
		return nil, err
	}

	return saved, nil
}

func (s *Store) SyncShifts(shifts []model.Shift) ([]model.Shift, int64, error) {
	sourceKeys := make([]string, 0, len(shifts))
	for _, shift := range shifts {
		sourceKeys = append(sourceKeys, model.ShiftSourceKey(shift))
	}

	var saved []model.Shift
	var deletedCount int64

	err := s.db.Transaction(func(tx *gorm.DB) error {
		store := &Store{db: tx}
		var err error
		saved, err = store.upsertShifts(shifts)
		if err != nil {
			return err
		}

		query := tx.Model(&ShiftRecord{})
		if len(sourceKeys) > 0 {
			query = query.Where("source_key NOT IN ?", sourceKeys)
		}

		result := query.Delete(&ShiftRecord{})
		if result.Error != nil {
			return fmt.Errorf("Excelから消えたshiftの削除に失敗しました: %w", result.Error)
		}

		deletedCount = result.RowsAffected
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	return saved, deletedCount, nil
}

func (s *Store) upsertShifts(shifts []model.Shift) ([]model.Shift, error) {
	saved := make([]model.Shift, 0, len(shifts))

	for _, shift := range shifts {
		sourceKey := model.ShiftSourceKey(shift)
		contentHash := model.ShiftContentHash(shift)

		var record ShiftRecord
		err := s.db.Where("source_key = ?", sourceKey).First(&record).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			record = ShiftRecord{
				StaffName:   shift.StaffName,
				StartTime:   shift.StartTime,
				EndTime:     shift.EndTime,
				Location:    shift.Location,
				SourceKey:   sourceKey,
				ContentHash: contentHash,
			}
			if err := s.db.Create(&record).Error; err != nil {
				return nil, fmt.Errorf("shiftの作成に失敗しました: %w", err)
			}
		case err != nil:
			return nil, fmt.Errorf("shiftの検索に失敗しました: %w", err)
		case record.ContentHash != contentHash:
			record.StaffName = shift.StaffName
			record.StartTime = shift.StartTime
			record.EndTime = shift.EndTime
			record.Location = shift.Location
			record.ContentHash = contentHash
			if err := s.db.Save(&record).Error; err != nil {
				return nil, fmt.Errorf("shiftの更新に失敗しました: %w", err)
			}
		}

		saved = append(saved, shiftFromRecord(record))
	}

	return saved, nil
}

func (s *Store) BeginShiftImport(filePath string, now time.Time) (model.ShiftImport, error) {
	return s.beginShiftImportWithSource(filePath, "local_excel", now)
}

func (s *Store) BeginLineShiftImport(filePath string, now time.Time) (model.ShiftImport, error) {
	return s.beginShiftImportWithSource(filePath, "line_excel", now)
}

func (s *Store) beginShiftImportWithSource(filePath string, sourceType string, now time.Time) (model.ShiftImport, error) {
	fileHash, err := fileSHA256(filePath)
	if err != nil {
		return model.ShiftImport{}, fmt.Errorf("Excelファイルのハッシュ計算に失敗しました: %w", err)
	}

	record := ShiftImportRecord{
		SourceType: sourceType,
		FileName:   filepath.Base(filePath),
		FilePath:   filePath,
		FileHash:   fileHash,
		Status:     string(model.ShiftImportStatusParsing),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.db.Create(&record).Error; err != nil {
		return model.ShiftImport{}, fmt.Errorf("取込履歴の作成に失敗しました: %w", err)
	}

	return shiftImportFromRecord(record), nil
}

func (s *Store) MarkShiftImportImported(id uint, now time.Time) error {
	return s.db.Model(&ShiftImportRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"imported_at":   &now,
			"status":        string(model.ShiftImportStatusImported),
			"error_message": "",
			"updated_at":    now,
		}).Error
}

func (s *Store) MarkShiftImportFailed(id uint, message string, now time.Time) error {
	return s.db.Model(&ShiftImportRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        string(model.ShiftImportStatusFailed),
			"error_message": message,
			"updated_at":    now,
		}).Error
}

func (s *Store) FindPendingNotificationTargets(now time.Time, before time.Duration) ([]model.Shift, error) {
	var records []ShiftRecord
	if err := s.db.
		Where("start_time > ? AND start_time <= ?", now, now.Add(before)).
		Order("start_time ASC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("通知対象shiftの検索に失敗しました: %w", err)
	}

	targets := make([]model.Shift, 0, len(records))
	for _, record := range records {
		shift := shiftFromRecord(record)
		if !s.AlreadyNotified(shift, model.NotificationTypeOneHourBefore) {
			targets = append(targets, shift)
		}
	}

	return targets, nil
}

func (s *Store) AlreadyNotified(shift model.Shift, notificationType model.NotificationType) bool {
	shiftID, ok := s.resolveShiftID(shift)
	if !ok {
		return false
	}

	var count int64
	err := s.db.Model(&ShiftNotificationRecord{}).
		Where(
			"shift_id = ? AND notification_type = ? AND status IN ?",
			shiftID,
			string(notificationType),
			[]string{
				string(model.NotificationStatusPending),
				string(model.NotificationStatusSent),
			},
		).
		Count(&count).Error
	return err == nil && count > 0
}

func (s *Store) SaveNotification(notification model.ShiftNotification) error {
	shiftID, ok := s.resolveShiftID(notification.Shift)
	if !ok {
		return fmt.Errorf("通知対象shiftが見つかりません: %s", model.ShiftSourceKey(notification.Shift))
	}

	record := notificationRecordFromModel(notification, shiftID)
	return s.db.Save(&record).Error
}

func (s *Store) resolveShiftID(shift model.Shift) (uint, bool) {
	if shift.ID != 0 {
		return shift.ID, true
	}

	var record ShiftRecord
	err := s.db.Where("source_key = ?", model.ShiftSourceKey(shift)).First(&record).Error
	if err != nil {
		return 0, false
	}

	return record.ID, true
}

func shiftFromRecord(record ShiftRecord) model.Shift {
	return model.Shift{
		ID:          record.ID,
		StaffName:   record.StaffName,
		StartTime:   record.StartTime,
		EndTime:     record.EndTime,
		Location:    record.Location,
		SourceKey:   record.SourceKey,
		ContentHash: record.ContentHash,
	}
}

func notificationRecordFromModel(notification model.ShiftNotification, shiftID uint) ShiftNotificationRecord {
	id := notification.ID
	if id == "" {
		id = scheduler.NotificationID(notification.Shift, notification.NotificationType)
	}

	return ShiftNotificationRecord{
		ID:               id,
		ShiftID:          shiftID,
		NotificationType: string(notification.NotificationType),
		ScheduledFor:     notification.ScheduledFor,
		SentAt:           notification.SentAt,
		Status:           string(notification.Status),
		ErrorMessage:     notification.ErrorMessage,
		CreatedAt:        notification.CreatedAt,
		UpdatedAt:        notification.UpdatedAt,
	}
}

func shiftImportFromRecord(record ShiftImportRecord) model.ShiftImport {
	return model.ShiftImport{
		ID:           record.ID,
		SourceType:   record.SourceType,
		FileName:     record.FileName,
		FilePath:     record.FilePath,
		FileHash:     record.FileHash,
		ImportedAt:   record.ImportedAt,
		Status:       model.ShiftImportStatus(record.Status),
		ErrorMessage: record.ErrorMessage,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
