package notification

import (
	"fmt"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

func BuildShiftReminderMessage(shift model.Shift, reminderBefore time.Duration) string {
	return fmt.Sprintf(
		"勤務開始%s前です。\n\n講師: %s\n場所: %s\n時間: %s〜%s",
		formatDuration(reminderBefore),
		shift.StaffName,
		shift.Location,
		formatDateTime(shift.StartTime),
		shift.EndTime.Format("15:04"),
	)
}

func formatDateTime(t time.Time) string {
	return t.Format("2006/01/02 15:04")
}

func formatDuration(d time.Duration) string {
	if d%time.Hour == 0 {
		return fmt.Sprintf("%d時間", int(d/time.Hour))
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%d分", int(d/time.Minute))
	}
	return d.String()
}
