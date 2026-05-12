package notification

import (
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

type Sender interface {
	SendShiftReminder(shift model.Shift, message string) error
}

func SendShiftReminder(
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

	message := BuildShiftReminderMessage(planned.Shift)
	if err := sender.SendShiftReminder(planned.Shift, message); err != nil {
		planned.Status = model.NotificationStatusFailed
		planned.ErrorMessage = err.Error()
		return planned
	}

	planned.Status = model.NotificationStatusSent
	planned.SentAt = &now
	planned.ErrorMessage = ""
	return planned
}
