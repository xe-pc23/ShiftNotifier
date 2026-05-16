package repository

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
	"github.com/xe-pc23/shift-notifier/internal/scheduler"
)

func TestUpsertShiftsCreatesAndUpdates(t *testing.T) {
	store := newTestStore(t)

	shift := model.Shift{
		StaffName: "柴田",
		StartTime: time.Date(2026, 5, 16, 18, 0, 0, 0, time.Local),
		EndTime:   time.Date(2026, 5, 16, 20, 0, 0, 0, time.Local),
		Location:  "A教室",
	}

	saved, err := store.UpsertShifts([]model.Shift{shift})
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 1 {
		t.Fatalf("len(saved) = %d, want 1", len(saved))
	}
	if saved[0].ID == 0 {
		t.Fatal("saved[0].ID is zero")
	}

	updated := shift
	updated.EndTime = time.Date(2026, 5, 16, 21, 0, 0, 0, time.Local)
	updated.SourceKey = model.ShiftSourceKey(shift)

	saved, err = store.UpsertShifts([]model.Shift{updated})
	if err != nil {
		t.Fatal(err)
	}

	if saved[0].ID == 0 {
		t.Fatal("updated shift ID is zero")
	}
	if !saved[0].EndTime.Equal(updated.EndTime) {
		t.Fatalf("EndTime = %s, want %s", saved[0].EndTime, updated.EndTime)
	}
}

func TestFindPendingNotificationTargetsSkipsNotifiedShift(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 5, 16, 17, 0, 0, 0, time.Local)

	notifyTarget := model.Shift{
		StaffName: "通知対象",
		StartTime: time.Date(2026, 5, 16, 18, 0, 0, 0, time.Local),
		EndTime:   time.Date(2026, 5, 16, 20, 0, 0, 0, time.Local),
		Location:  "A教室",
	}
	notified := model.Shift{
		StaffName: "通知済み",
		StartTime: time.Date(2026, 5, 16, 18, 30, 0, 0, time.Local),
		EndTime:   time.Date(2026, 5, 16, 20, 30, 0, 0, time.Local),
		Location:  "B教室",
	}

	saved, err := store.UpsertShifts([]model.Shift{notifyTarget, notified})
	if err != nil {
		t.Fatal(err)
	}

	planned := model.ShiftNotification{
		ID:               scheduler.NotificationID(saved[1], model.NotificationTypeOneHourBefore),
		Shift:            saved[1],
		NotificationType: model.NotificationTypeOneHourBefore,
		ScheduledFor:     saved[1].StartTime.Add(-time.Hour),
		Status:           model.NotificationStatusSent,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := store.SaveNotification(planned); err != nil {
		t.Fatal(err)
	}

	targets, err := store.FindPendingNotificationTargets(now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if len(targets) != 1 {
		t.Fatalf("len(targets) = %d, want 1", len(targets))
	}
	if targets[0].StaffName != "通知対象" {
		t.Fatalf("targets[0].StaffName = %q, want %q", targets[0].StaffName, "通知対象")
	}
}

func TestShiftImportLifecycle(t *testing.T) {
	store := newTestStore(t)
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.Local)
	filePath := filepath.Join(t.TempDir(), "shift.xlsx")
	if err := os.WriteFile(filePath, []byte("excel bytes"), 0600); err != nil {
		t.Fatal(err)
	}

	shiftImport, err := store.BeginShiftImport(filePath, now)
	if err != nil {
		t.Fatal(err)
	}

	if shiftImport.ID == 0 {
		t.Fatal("shiftImport.ID is zero")
	}
	if shiftImport.Status != model.ShiftImportStatusParsing {
		t.Fatalf("Status = %q, want %q", shiftImport.Status, model.ShiftImportStatusParsing)
	}
	if shiftImport.FileHash == "" {
		t.Fatal("FileHash is empty")
	}

	importedAt := now.Add(time.Minute)
	if err := store.MarkShiftImportImported(shiftImport.ID, importedAt); err != nil {
		t.Fatal(err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := OpenSQLite(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}

	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}

	return NewStore(db)
}
