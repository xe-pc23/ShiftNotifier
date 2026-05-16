package main

import (
	"testing"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/scheduler"
)

func TestShiftReminderBeforeDefault(t *testing.T) {
	t.Setenv("SHIFT_NOTIFIER_REMINDER_BEFORE", "")

	got, err := shiftReminderBefore()
	if err != nil {
		t.Fatal(err)
	}

	if got != scheduler.DefaultShiftReminderBefore {
		t.Fatalf("shiftReminderBefore() = %s, want %s", got, scheduler.DefaultShiftReminderBefore)
	}
}

func TestShiftReminderBeforeFromEnv(t *testing.T) {
	t.Setenv("SHIFT_NOTIFIER_REMINDER_BEFORE", "30m")

	got, err := shiftReminderBefore()
	if err != nil {
		t.Fatal(err)
	}

	if got != 30*time.Minute {
		t.Fatalf("shiftReminderBefore() = %s, want 30m", got)
	}
}

func TestShiftReminderBeforeRejectsInvalidValue(t *testing.T) {
	t.Setenv("SHIFT_NOTIFIER_REMINDER_BEFORE", "soon")

	if _, err := shiftReminderBefore(); err == nil {
		t.Fatal("err = nil, want error")
	}
}
