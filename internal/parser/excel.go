package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/model"

	"github.com/xuri/excelize/v2"
)

type SourceConfig struct {
	StaffHeaderRow int
	DataStartRow   int
	DateColumn     string
	TargetSheet    string
	StaffBlocks    []StaffBlock
}

type StaffBlock struct {
	NameCol      string
	StartTimeCol string
	EndTimeCol   string
	LocationCol  string
}

var DefaultSourceConfig = SourceConfig{
	StaffHeaderRow: 2,
	DataStartRow:   1,
	DateColumn:     "A",
	TargetSheet:    "柴田君送付用",
	StaffBlocks: []StaffBlock{
		{NameCol: "B", StartTimeCol: "B", EndTimeCol: "C", LocationCol: "D"},
		{NameCol: "F", StartTimeCol: "F", EndTimeCol: "G", LocationCol: "H"},
		{NameCol: "J", StartTimeCol: "J", EndTimeCol: "K", LocationCol: "L"},
		{NameCol: "N", StartTimeCol: "N", EndTimeCol: "O", LocationCol: "P"},
		{NameCol: "R", StartTimeCol: "R", EndTimeCol: "S", LocationCol: "T"},
		{NameCol: "V", StartTimeCol: "V", EndTimeCol: "W", LocationCol: "X"},
		{NameCol: "Z", StartTimeCol: "Z", EndTimeCol: "AA", LocationCol: "AB"},
	},
}

func ParseExcel(path string, config SourceConfig) ([]model.Shift, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("Excelファイルを開けませんでした: %w", err)
	}
	defer f.Close()

	sheetName := config.TargetSheet

	sheetIndex, err := f.GetSheetIndex(sheetName) //指定したシートが存在するかどうか
	if err != nil {
		return nil, fmt.Errorf("シート %q の確認に失敗しました: %w", sheetName, err)
	}

	if sheetIndex == -1 { //シートが見つからない場合は-1が返るライブラリの仕様
		return nil, fmt.Errorf("シート %q が見つかりませんでした", sheetName)
	}

	rows, err := f.GetRows(sheetName) // 行数確認の為
	if err != nil {
		return nil, fmt.Errorf("シートの読み込みに失敗しました: %w", err)
	}

	staffNames, err := readStaffNames(f, sheetName, config)
	if err != nil {
		return nil, err
	}

	var shifts []model.Shift

	var currentYear int
	var currentMonth int

	for rowIndex := config.DataStartRow; rowIndex <= len(rows); rowIndex++ {
		dateText, err := getCellValue(f, sheetName, config.DateColumn, rowIndex)
		if err != nil {
			return nil, err
		}

		dateText = strings.TrimSpace(dateText) // 前後の空白を削除
		if dateText == "" {
			continue
		}

		year, month, err := extractYearMonth(dateText)
		if err == nil {
			currentYear = year
			currentMonth = month
			continue
		}

		date, ok := parseDateText(dateText, currentYear, currentMonth)
		if !ok {
			continue
		}

		for i, block := range config.StaffBlocks {
			staffName := staffNames[i]
			if staffName == "" {
				continue
			}

			shift, ok, err := readShiftFromBlock(f, sheetName, rowIndex, block, staffName, date)
			if err != nil {
				return nil, err
			}

			if ok {
				shifts = append(shifts, shift)
			}
		}
	}

	return uniqueShifts(shifts), nil
}

func readStaffNames(f *excelize.File, sheetName string, config SourceConfig) ([]string, error) {
	staffNames := make([]string, len(config.StaffBlocks))

	for i, block := range config.StaffBlocks {
		name, err := getCellValue(f, sheetName, block.NameCol, config.StaffHeaderRow)
		if err != nil {
			return nil, err
		}

		staffNames[i] = strings.TrimSpace(name) // 前後の空白を削除して保存
	}

	return staffNames, nil
}

func readShiftFromBlock(
	f *excelize.File,
	sheetName string,
	rowIndex int,
	block StaffBlock,
	staffName string,
	currentDate time.Time,
) (model.Shift, bool, error) {
	startTimeText, err := getCellValue(f, sheetName, block.StartTimeCol, rowIndex)
	if err != nil {
		return model.Shift{}, false, err
	}

	endTimeText, err := getCellValue(f, sheetName, block.EndTimeCol, rowIndex)
	if err != nil {
		return model.Shift{}, false, err
	}

	location, err := getCellValue(f, sheetName, block.LocationCol, rowIndex)
	if err != nil {
		return model.Shift{}, false, err
	}

	startTimeText = strings.TrimSpace(startTimeText)
	endTimeText = strings.TrimSpace(endTimeText)
	location = strings.TrimSpace(location)

	if startTimeText == "" || endTimeText == "" || location == "" {
		return model.Shift{}, false, nil
	}

	if isTimeLike(location) {
		return model.Shift{}, false, nil
	}

	startHour, startMinute, err := parseTimeText(startTimeText)
	if err != nil {
		return model.Shift{}, false, nil
	}

	endHour, endMinute, err := parseTimeText(endTimeText)
	if err != nil {
		return model.Shift{}, false, nil
	}

	startTime := time.Date(
		currentDate.Year(),
		currentDate.Month(),
		currentDate.Day(),
		startHour,
		startMinute,
		0,
		0,
		time.Local,
	)

	endTime := time.Date(
		currentDate.Year(),
		currentDate.Month(),
		currentDate.Day(),
		endHour,
		endMinute,
		0,
		0,
		time.Local,
	)

	return model.Shift{
		StaffName: staffName,
		StartTime: startTime,
		EndTime:   endTime,
		Location:  location,
	}, true, nil
}

func getCellValue(f *excelize.File, sheetName string, col string, row int) (string, error) {
	cell, err := excelize.JoinCellName(col, row) // 例）AA 10 -> AA10
	if err != nil {
		return "", err
	}

	value, err := f.GetCellValue(sheetName, cell)
	if err != nil {
		return "", err
	}

	return value, nil
}

func parseDateText(text string, currentYear int, currentMonth int) (time.Time, bool) {
	text = normalizeNumber(strings.TrimSpace(text))

	if serial, err := strconv.ParseFloat(text, 64); err == nil {
		date, err := excelize.ExcelDateToTime(serial, false)
		if err == nil {
			return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local), true
		}
	}

	layouts := []string{
		"2006/1/2",
		"2006/01/02",
		"2006-1-2",
		"2006-01-02",
		"1/2/06",
		"1/2/2006",
	}

	for _, layout := range layouts {
		date, err := time.ParseInLocation(layout, text, time.Local)
		if err == nil {
			return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local), true
		}
	}

	if currentYear != 0 && currentMonth != 0 {
		day, err := extractDay(text)
		if err == nil {
			date := time.Date(currentYear, time.Month(currentMonth), day, 0, 0, 0, 0, time.Local)
			return date, true
		}
	}

	return time.Time{}, false
}

func extractYearMonth(text string) (int, int, error) {
	text = normalizeNumber(text)//全角数字を半角に変換

	re := regexp.MustCompile(`([0-9]{4})\s*年\s*([0-9]{1,2})\s*月`)//正規表現して*regexp.Regexp を返す
	matches := re.FindStringSubmatch(text)

	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("年月を取得できませんでした: %s", text)
	}

	year, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}

	month, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, err
	}

	if month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("月の値が不正です: %d", month)
	}

	return year, month, nil
}

func extractDay(text string) (int, error) {
	text = normalizeNumber(strings.TrimSpace(text))

	re := regexp.MustCompile(`^([0-9]{1,2})`)
	matches := re.FindStringSubmatch(text)

	if len(matches) < 2 {
		return 0, fmt.Errorf("日付を取得できませんでした: %s", text)
	}

	day, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	if day < 1 || day > 31 {
		return 0, fmt.Errorf("日付の値が不正です: %d", day)
	}

	return day, nil
}

func parseTimeText(text string) (int, int, error) {
	text = normalizeNumber(strings.TrimSpace(text))
	text = strings.ReplaceAll(text, "：", ":")

	if serial, err := strconv.ParseFloat(text, 64); err == nil {
		if serial <= 0 || serial >= 1 {
			return 0, 0, fmt.Errorf("時刻の数値が不正です: %s", text)
		}

		totalMinutes := int(serial*24*60 + 0.5)
		hour := totalMinutes / 60
		minute := totalMinutes % 60

		if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			return 0, 0, fmt.Errorf("時刻の値が不正です: %s", text)
		}

		return hour, minute, nil
	}

	re := regexp.MustCompile(`([0-9]{1,2}):([0-9]{2})`)
	matches := re.FindStringSubmatch(text)

	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("時刻形式が不正です: %s", text)
	}

	hour, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}

	minute, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, err
	}

	if hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("時が不正です: %d", hour)
	}

	if minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("分が不正です: %d", minute)
	}

	return hour, minute, nil
}

func isTimeLike(text string) bool {
	text = normalizeNumber(strings.TrimSpace(text))
	text = strings.ReplaceAll(text, "：", ":")

	_, _, err := parseTimeText(text)
	return err == nil
}

func uniqueShifts(shifts []model.Shift) []model.Shift {
	seen := make(map[string]bool)
	var result []model.Shift

	for _, shift := range shifts {
		key := shift.StaffName + "|" +
			shift.StartTime.Format(time.RFC3339) + "|" +
			shift.EndTime.Format(time.RFC3339) + "|" +
			shift.Location

		if seen[key] {
			continue
		}

		seen[key] = true
		result = append(result, shift)
	}

	return result
}

func normalizeNumber(s string) string {
	replacer := strings.NewReplacer(
		"０", "0",
		"１", "1",
		"２", "2",
		"３", "3",
		"４", "4",
		"５", "5",
		"６", "6",
		"７", "7",
		"８", "8",
		"９", "9",
	)

	return replacer.Replace(s)
}
