package model

import "time"

type NotificationType string

const (
	NotificationTypeShiftReminder NotificationType = "shift_reminder"
)

type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

// タイポや意図を明確にするために型で定義した

type ShiftNotification struct {
	ID               string
	Shift            Shift
	NotificationType NotificationType
	ScheduledFor     time.Time
	SentAt           *time.Time
	Status           NotificationStatus
	ErrorMessage     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
