package scheduler

import (
	"testing"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
)

func TestShouldNotify(t *testing.T) {
	shift := model.Shift{
		StaffName: "柴田",
		StartTime: time.Date(2026, 5, 8, 18, 0, 0, 0, time.Local),
		EndTime:   time.Date(2026, 5, 8, 20, 0, 0, 0, time.Local),
		Location:  "A教室",
	}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{
			name: "1時間前ちょうどなら通知対象",
			now:  time.Date(2026, 5, 8, 17, 0, 0, 0, time.Local),
			want: true,
		},
		{
			name: "1時間前を過ぎて開始前なら通知対象",
			now:  time.Date(2026, 5, 8, 17, 30, 0, 0, time.Local),
			want: true,
		},
		{
			name: "1時間前より前なら通知しない",
			now:  time.Date(2026, 5, 8, 16, 59, 0, 0, time.Local),
			want: false,
		},
		{
			name: "開始時刻を過ぎたら通知しない",
			now:  time.Date(2026, 5, 8, 18, 0, 0, 0, time.Local),
			want: false,
		},
		{
			name: "日付が違っても1時間前より前なら通知しない",
			now:  time.Date(2026, 5, 7, 23, 30, 0, 0, time.Local),
			want: false,
		},
		{
			name: "開始後なら過去の日付として通知しない",
			now:  time.Date(2026, 5, 9, 17, 30, 0, 0, time.Local),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldNotify(shift, tt.now, time.Hour)
			if got != tt.want {
				t.Fatalf("ShouldNotify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldNotifyAcrossDateBoundary(t *testing.T) {
	shift := model.Shift{
		StaffName: "佐藤",
		StartTime: time.Date(2026, 5, 9, 0, 30, 0, 0, time.Local),
		EndTime:   time.Date(2026, 5, 9, 2, 30, 0, 0, time.Local),
		Location:  "深夜教室",
	}

	now := time.Date(2026, 5, 8, 23, 30, 0, 0, time.Local)

	if !ShouldNotify(shift, now, time.Hour) {
		t.Fatal("ShouldNotify() = false, want true")
	}
}

func TestFindNotifyTargets(t *testing.T) {
	now := time.Date(2026, 5, 8, 17, 0, 0, 0, time.Local)
	shifts := []model.Shift{
		{
			StaffName: "通知対象",
			StartTime: time.Date(2026, 5, 8, 18, 0, 0, 0, time.Local),
			EndTime:   time.Date(2026, 5, 8, 20, 0, 0, 0, time.Local),
			Location:  "A教室",
		},
		{
			StaffName: "未来すぎる",
			StartTime: time.Date(2026, 5, 8, 19, 0, 0, 0, time.Local),
			EndTime:   time.Date(2026, 5, 8, 21, 0, 0, 0, time.Local),
			Location:  "B教室",
		},
	}

	targets := FindNotifyTargets(shifts, now, time.Hour)

	if len(targets) != 1 {
		t.Fatalf("len(targets) = %d, want 1", len(targets))
	}

	if targets[0].StaffName != "通知対象" {
		t.Fatalf("targets[0].StaffName = %q, want %q", targets[0].StaffName, "通知対象")
	}
}
