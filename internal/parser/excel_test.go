package parser

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"
	"github.com/xuri/excelize/v2"
)

func TestParseExcelUpdatesStaffNamesWhenMonthlyHeaderChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shift.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	sheet := DefaultSourceConfig.TargetSheet
	f.SetSheetName("Sheet1", sheet)

	mustSetCell(t, f, sheet, "A1", "2026年4月")
	mustSetCell(t, f, sheet, "B2", "退職した人")
	mustSetCell(t, f, sheet, "A3", "1日")
	mustSetCell(t, f, sheet, "B3", "10:00")
	mustSetCell(t, f, sheet, "C3", "12:00")
	mustSetCell(t, f, sheet, "D3", "101")

	mustSetCell(t, f, sheet, "A4", "2026年5月")
	mustSetCell(t, f, sheet, "B5", "新しく入った人")
	mustSetCell(t, f, sheet, "A6", "1日")
	mustSetCell(t, f, sheet, "B6", "13:00")
	mustSetCell(t, f, sheet, "C6", "15:00")
	mustSetCell(t, f, sheet, "D6", "102")

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs() error = %v", err)
	}

	shifts, err := ParseExcel(path, DefaultSourceConfig)
	if err != nil {
		t.Fatalf("ParseExcel() error = %v", err)
	}

	if len(shifts) != 2 {
		t.Fatalf("len(shifts) = %d, want 2", len(shifts))
	}

	assertShift(t, shifts[0], "退職した人", time.Date(2026, 4, 1, 10, 0, 0, 0, time.Local), "101")
	assertShift(t, shifts[1], "新しく入った人", time.Date(2026, 5, 1, 13, 0, 0, 0, time.Local), "102")
}

func TestParseExcelUpdatesStaffNamesFromHeaderRowBeforeNextMonthTitle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shift.xlsx")

	f := excelize.NewFile()
	defer f.Close()

	sheet := DefaultSourceConfig.TargetSheet
	f.SetSheetName("Sheet1", sheet)

	mustSetCell(t, f, sheet, "A1", "2026年4月")
	mustSetCell(t, f, sheet, "B2", "退職した人")
	mustSetCell(t, f, sheet, "A3", "30日")
	mustSetCell(t, f, sheet, "B3", "10:00")
	mustSetCell(t, f, sheet, "C3", "12:00")
	mustSetCell(t, f, sheet, "D3", "101")

	mustSetCell(t, f, sheet, "B4", "新しく入った人")
	mustSetCell(t, f, sheet, "A5", "2026年5月")
	mustSetCell(t, f, sheet, "A6", "1日")
	mustSetCell(t, f, sheet, "B6", "13:00")
	mustSetCell(t, f, sheet, "C6", "15:00")
	mustSetCell(t, f, sheet, "D6", "102")

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs() error = %v", err)
	}

	shifts, err := ParseExcel(path, DefaultSourceConfig)
	if err != nil {
		t.Fatalf("ParseExcel() error = %v", err)
	}

	if len(shifts) != 2 {
		t.Fatalf("len(shifts) = %d, want 2", len(shifts))
	}

	assertShift(t, shifts[0], "退職した人", time.Date(2026, 4, 30, 10, 0, 0, 0, time.Local), "101")
	assertShift(t, shifts[1], "新しく入った人", time.Date(2026, 5, 1, 13, 0, 0, 0, time.Local), "102")
}

func mustSetCell(t *testing.T, f *excelize.File, sheet string, cell string, value any) {
	t.Helper()

	if err := f.SetCellValue(sheet, cell, value); err != nil {
		t.Fatalf("SetCellValue(%s) error = %v", cell, err)
	}
}

func assertShift(t *testing.T, shift model.Shift, staffName string, startTime time.Time, location string) {
	t.Helper()

	if shift.StaffName != staffName {
		t.Fatalf("StaffName = %q, want %q", shift.StaffName, staffName)
	}

	if !shift.StartTime.Equal(startTime) {
		t.Fatalf("StartTime = %s, want %s", shift.StartTime, startTime)
	}

	if shift.Location != location {
		t.Fatalf("Location = %q, want %q", shift.Location, location)
	}
}
