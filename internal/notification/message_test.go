package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

func TestBuildShiftReminderMessage(t *testing.T) {
	shift := model.Shift{
		StaffName: "柴田",
		StartTime: time.Date(2026, 5, 8, 18, 0, 0, 0, time.Local),
		EndTime:   time.Date(2026, 5, 8, 20, 0, 0, 0, time.Local),
		Location:  "A教室",
	}

	message := BuildShiftReminderMessage(shift)

	wants := []string{
		"まもなく勤務開始です。",
		"講師: 柴田",
		"場所: A教室",
		"時間: 2026/05/08 18:00〜20:00",
	}

	for _, want := range wants {
		if !strings.Contains(message, want) {
			t.Fatalf("message does not contain %q:\n%s", want, message)
		}
	}
}
