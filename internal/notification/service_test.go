package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

func TestSendShiftReminderMarksSent(t *testing.T) {
	now := time.Date(2026, 5, 8, 17, 0, 0, 0, time.Local)
	planned := model.ShiftNotification{
		Shift: model.Shift{
			StaffName: "柴田",
			StartTime: time.Date(2026, 5, 8, 18, 0, 0, 0, time.Local),
			EndTime:   time.Date(2026, 5, 8, 20, 0, 0, 0, time.Local),
			Location:  "A教室",
		},
		Status: model.NotificationStatusPending,
	}

	sender := &fakeSender{}
	sent := SendShiftReminder(context.Background(), planned, sender, now)

	if sent.Status != model.NotificationStatusSent {
		t.Fatalf("Status = %q, want %q", sent.Status, model.NotificationStatusSent)
	}

	if sent.SentAt == nil || !sent.SentAt.Equal(now) {
		t.Fatalf("SentAt = %v, want %s", sent.SentAt, now)
	}

	if sender.message == "" {
		t.Fatal("sender.message is empty")
	}
}

func TestSendShiftReminderMarksFailed(t *testing.T) {
	now := time.Date(2026, 5, 8, 17, 0, 0, 0, time.Local)
	planned := model.ShiftNotification{
		Shift: model.Shift{
			StaffName: "柴田",
			StartTime: time.Date(2026, 5, 8, 18, 0, 0, 0, time.Local),
			EndTime:   time.Date(2026, 5, 8, 20, 0, 0, 0, time.Local),
			Location:  "A教室",
		},
		Status: model.NotificationStatusPending,
	}

	sent := SendShiftReminder(context.Background(), planned, &fakeSender{err: errors.New("line api failed")}, now)

	if sent.Status != model.NotificationStatusFailed {
		t.Fatalf("Status = %q, want %q", sent.Status, model.NotificationStatusFailed)
	}

	if sent.ErrorMessage != "line api failed" {
		t.Fatalf("ErrorMessage = %q, want %q", sent.ErrorMessage, "line api failed")
	}
}

type fakeSender struct {
	message string
	err     error
}

func (s *fakeSender) SendShiftReminder(ctx context.Context, shift model.Shift, message string) error {
	s.message = message
	return s.err
}

func TestParseStaffRecipients(t *testing.T) {
	recipients, err := ParseStaffRecipients("柴田=U123, 佐藤=U456")
	if err != nil {
		t.Fatal(err)
	}

	if recipients["柴田"] != "U123" {
		t.Fatalf("recipients[柴田] = %q, want %q", recipients["柴田"], "U123")
	}

	if recipients["佐藤"] != "U456" {
		t.Fatalf("recipients[佐藤] = %q, want %q", recipients["佐藤"], "U456")
	}
}

func TestLineSenderRequiresRecipient(t *testing.T) {
	sender := NewLineSender(&fakePushClient{}, map[string]string{})
	err := sender.SendShiftReminder(
		context.Background(),
		model.Shift{StaffName: "未設定"},
		"message",
	)

	if err == nil {
		t.Fatal("err = nil, want error")
	}
}

type fakePushClient struct{}

func (c *fakePushClient) PushText(ctx context.Context, to string, text string) error {
	return nil
}
