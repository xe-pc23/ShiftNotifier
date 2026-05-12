package scheduler

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

type NotificationHistory interface {
	AlreadyNotified(shift model.Shift, notificationType model.NotificationType) bool
}

func FindNotifyTargets(shifts []model.Shift, now time.Time, before time.Duration) []model.Shift {
	var targets []model.Shift

	for _, shift := range shifts {
		if ShouldNotify(shift, now, before) {
			targets = append(targets, shift)
		}
	}

	return targets
}

func ShouldNotify(shift model.Shift, now time.Time, before time.Duration) bool {
	if before <= 0 {
		return false
	}

	notifyTime := shift.StartTime.Add(-before)

	if now.Before(notifyTime) {
		return false
	}

	if !now.Before(shift.StartTime) {
		return false
	}

	return true // 通知対象とする
}

func PlanShiftNotifications(
	shifts []model.Shift,
	now time.Time,
	before time.Duration,
	history NotificationHistory,
) []model.ShiftNotification {
	targets := FindNotifyTargets(shifts, now, before)
	notifications := make([]model.ShiftNotification, 0, len(targets))

	for _, shift := range targets {
		notificationType := model.NotificationTypeOneHourBefore
		if history != nil && history.AlreadyNotified(shift, notificationType) {
			continue
		}

		notifications = append(notifications, model.ShiftNotification{
			ID:               NotificationID(shift, notificationType),
			Shift:            shift,
			NotificationType: notificationType,
			ScheduledFor:     shift.StartTime.Add(-before),
			Status:           model.NotificationStatusPending,
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}

	return notifications
}

func NotificationID(shift model.Shift, notificationType model.NotificationType) string {
	key := fmt.Sprintf(
		"%s|%s|%s|%s|%s",
		notificationType,
		shift.StaffName,
		shift.StartTime.Format(time.RFC3339),
		shift.EndTime.Format(time.RFC3339),
		shift.Location,
	)

	sum := sha1.Sum([]byte(key))      // SHA-1はセキュリティ目的ではなく、簡単に一意のIDを生成するために使用
	return hex.EncodeToString(sum[:]) // 16進数の文字列に変換して返す
}
