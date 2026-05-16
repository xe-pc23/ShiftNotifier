package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/notification"
	"github.com/xe-pc23/shift-notifier/internal/parser"
	"github.com/xe-pc23/shift-notifier/internal/repository"
	"github.com/xe-pc23/shift-notifier/internal/scheduler"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 { //　引数が設定されているのかの確認　文字列の数で判断
		fmt.Println("使い方:")
		fmt.Println("  go run ./cmd/shift-notifier ./testdata/shift.xlsx")
		os.Exit(1)
	}

	filePath := os.Args[1]
	dbPath := os.Getenv("SHIFT_NOTIFIER_DB_PATH")
	if dbPath == "" {
		dbPath = "shift_notifier.db"
	}

	db, err := repository.OpenSQLite(dbPath)
	if err != nil {
		return err
	}

	if err := repository.AutoMigrate(db); err != nil {
		return err
	}

	store := repository.NewStore(db)
	now := time.Now()

	shiftImport, err := store.BeginShiftImport(filePath, now)
	if err != nil {
		return err
	}
	fmt.Printf("取込履歴ID: %d\n", shiftImport.ID)
	fmt.Printf("ファイルハッシュ: %s\n\n", shiftImport.FileHash)

	shifts, err := parser.ParseExcel(filePath, parser.DefaultSourceConfig)
	if err != nil {
		_ = store.MarkShiftImportFailed(shiftImport.ID, err.Error(), time.Now())
		return err
	}

	fmt.Printf("読み込んだシフト数: %d件\n\n", len(shifts))
	savedShifts, err := store.UpsertShifts(shifts)
	if err != nil {
		_ = store.MarkShiftImportFailed(shiftImport.ID, err.Error(), time.Now())
		return err
	}
	fmt.Printf("DB保存済みシフト数: %d件\n\n", len(savedShifts))

	if err := store.MarkShiftImportImported(shiftImport.ID, time.Now()); err != nil {
		return err
	}

	targets, err := store.FindPendingNotificationTargets(now, time.Hour)
	if err != nil {
		return err
	}

	notifications := scheduler.PlanShiftNotifications(targets, now, time.Hour, store)

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

		if err := store.SaveNotification(planned); err != nil {
			return err
		}
	}

	return nil
}
