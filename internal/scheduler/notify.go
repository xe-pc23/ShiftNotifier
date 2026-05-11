package scheduler

import (
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

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
