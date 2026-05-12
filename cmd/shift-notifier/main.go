package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/notification"
	"github.com/xe-pc23/shift-notifier/internal/parser"
	"github.com/xe-pc23/shift-notifier/internal/scheduler"
)

func main() {
	if len(os.Args) < 2 { //　引数が設定されているのかの確認　文字列の数で判断
		fmt.Println("使い方:")
		fmt.Println("  go run ./cmd/shift-notifier ./testdata/shift.xlsx")
		os.Exit(1)
	}

	filePath := os.Args[1]

	shifts, err := parser.ParseExcel(filePath, parser.DefaultSourceConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("読み込んだシフト数: %d件\n\n", len(shifts))
	now := time.Now()

	notifications := scheduler.PlanShiftNotifications(shifts, now, time.Hour, nil)

	fmt.Printf("現在時刻: %s\n", now.Format("2006/01/02 15:04"))
	fmt.Printf("通知予定のシフト数: %d件\n\n", len(notifications))

	for _, planned := range notifications { //indexは使わないので_で無視
		shift := planned.Shift
		fmt.Printf(
			"通知予定: ID: %s / 講師: %s / 時間: %s〜%s / 場所: %s\n",
			planned.ID,
			shift.StaffName,
			shift.StartTime.Format("2006/01/02 15:04"),
			shift.EndTime.Format("15:04"),
			shift.Location,
		)
		fmt.Printf("メッセージ:\n%s\n\n", notification.BuildShiftReminderMessage(shift))
	}
}
