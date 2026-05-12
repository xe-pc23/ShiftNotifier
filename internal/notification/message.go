package notification

import (
	"fmt"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

func BuildShiftReminderMessage(shift model.Shift) string {
	return fmt.Sprintf(
		"まもなく勤務開始です。\n\n講師: %s\n場所: %s\n時間: %s〜%s",
		shift.StaffName,
		shift.Location,
		formatDateTime(shift.StartTime),
		shift.EndTime.Format("15:04"),
	)
}

func formatDateTime(t time.Time) string {
	return t.Format("2006/01/02 15:04")
}
