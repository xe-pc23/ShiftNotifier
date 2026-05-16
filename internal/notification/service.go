package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

type Sender interface {
	SendShiftReminder(ctx context.Context, shift model.Shift, message string) error
}

type PushClient interface {
	PushText(ctx context.Context, to string, text string) error
}

type LineSender struct {
	client     PushClient
	recipients map[string]string
}

func NewLineSender(client PushClient, recipients map[string]string) *LineSender {
	return &LineSender{
		client:     client,
		recipients: recipients,
	}
}

func (s *LineSender) SendShiftReminder(ctx context.Context, shift model.Shift, message string) error {
	if s.client == nil {
		return fmt.Errorf("LINE client is nil")
	}

	to := s.recipients[shift.StaffName]
	if to == "" {
		return fmt.Errorf("LINE送信先が未設定です: %s", shift.StaffName)
	}

	return s.client.PushText(ctx, to, message)
}

func ParseStaffRecipients(value string) (map[string]string, error) {
	recipients := make(map[string]string)
	value = strings.TrimSpace(value)
	if value == "" {
		return recipients, nil
	}

	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}

		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("LINE_STAFF_USER_IDS の形式が不正です: %s", item)
		}

		staffName := strings.TrimSpace(parts[0])
		userID := strings.TrimSpace(parts[1])
		if staffName == "" || userID == "" {
			return nil, fmt.Errorf("LINE_STAFF_USER_IDS の形式が不正です: %s", item)
		}

		recipients[staffName] = userID
	}

	return recipients, nil
}

func SendShiftReminder(
	ctx context.Context,
	planned model.ShiftNotification,
	sender Sender,
	now time.Time,
) model.ShiftNotification {
	planned.UpdatedAt = now

	if sender == nil {
		planned.Status = model.NotificationStatusFailed
		planned.ErrorMessage = "notification sender is nil"
		return planned
	}

	reminderBefore := planned.Shift.StartTime.Sub(planned.ScheduledFor)
	if planned.ScheduledFor.IsZero() {
		reminderBefore = planned.Shift.StartTime.Sub(now)
	}
	message := BuildShiftReminderMessage(planned.Shift, reminderBefore)
	if err := sender.SendShiftReminder(ctx, planned.Shift, message); err != nil {
		planned.Status = model.NotificationStatusFailed
		planned.ErrorMessage = err.Error()
		return planned
	}

	planned.Status = model.NotificationStatusSent
	planned.SentAt = &now
	planned.ErrorMessage = ""
	return planned
}
