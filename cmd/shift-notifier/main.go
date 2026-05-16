package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/linebot"
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
		fmt.Println("  go run ./cmd/shift-notifier run")
		fmt.Println("  go run ./cmd/shift-notifier serve")
		fmt.Println("  go run ./cmd/shift-notifier notify")
		fmt.Println("  go run ./cmd/shift-notifier notify-loop")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		return runAll(context.Background())
	case "serve":
		return runServer()
	case "notify":
		return runNotifyOnce(context.Background())
	case "notify-loop":
		return runNotifyLoop(context.Background())
	}

	store, err := openStore()
	if err != nil {
		return err
	}

	return importLocalExcel(store, os.Args[1])
}

func openStore() (*repository.Store, error) {
	dbPath := os.Getenv("SHIFT_NOTIFIER_DB_PATH")
	if dbPath == "" {
		dbPath = "shift_notifier.db"
	}

	db, err := repository.OpenSQLite(dbPath)
	if err != nil {
		return nil, err
	}

	if err := repository.AutoMigrate(db); err != nil {
		return nil, err
	}

	return repository.NewStore(db), nil
}

func importLocalExcel(store *repository.Store, filePath string) error {
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
	savedShifts, deletedCount, err := store.SyncShifts(shifts)
	if err != nil {
		_ = store.MarkShiftImportFailed(shiftImport.ID, err.Error(), time.Now())
		return err
	}
	fmt.Printf("DB保存済みシフト数: %d件\n\n", len(savedShifts))
	fmt.Printf("Excelから削除されたシフト数: %d件\n\n", deletedCount)

	if err := store.MarkShiftImportImported(shiftImport.ID, time.Now()); err != nil {
		return err
	}

	return nil
}

func runAll(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		errCh <- runServer()
	}()

	go func() {
		errCh <- runNotifyLoop(ctx)
	}()

	return <-errCh
}

func runServer() error {
	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
	if channelSecret == "" {
		return fmt.Errorf("LINE_CHANNEL_SECRET is required")
	}

	channelAccessToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if channelAccessToken == "" {
		return fmt.Errorf("LINE_CHANNEL_ACCESS_TOKEN is required")
	}

	importDir := os.Getenv("SHIFT_NOTIFIER_IMPORT_DIR")
	if importDir == "" {
		importDir = "imports"
	}

	addr := os.Getenv("SHIFT_NOTIFIER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	store, err := openStore()
	if err != nil {
		return err
	}

	client := linebot.NewClient(channelAccessToken)
	handler := linebot.NewWebhookHandler(channelSecret, client, store, importDir)

	mux := http.NewServeMux()
	mux.Handle("/webhook/line", handler)

	fmt.Printf("LINE webhook server listening on %s\n", addr)
	fmt.Println("Webhook path: /webhook/line")
	return http.ListenAndServe(addr, mux)
}

func runNotifyOnce(ctx context.Context) error {
	channelAccessToken := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	if channelAccessToken == "" {
		return fmt.Errorf("LINE_CHANNEL_ACCESS_TOKEN is required")
	}

	recipients, err := notification.ParseStaffRecipients(os.Getenv("LINE_STAFF_USER_IDS"))
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return fmt.Errorf("LINE_STAFF_USER_IDS is required")
	}

	store, err := openStore()
	if err != nil {
		return err
	}

	client := linebot.NewClient(channelAccessToken)
	sender := notification.NewLineSender(client, recipients)
	now := time.Now()

	targets, err := store.FindPendingNotificationTargets(now, time.Hour)
	if err != nil {
		return err
	}

	plannedNotifications := scheduler.PlanShiftNotifications(targets, now, time.Hour, store)
	fmt.Printf("現在時刻: %s\n", now.Format("2006/01/02 15:04"))
	fmt.Printf("通知対象のシフト数: %d件\n", len(plannedNotifications))

	for _, planned := range plannedNotifications {
		if err := store.SaveNotification(planned); err != nil {
			return err
		}

		sent := notification.SendShiftReminder(ctx, planned, sender, time.Now())
		if err := store.SaveNotification(sent); err != nil {
			return err
		}

		shift := sent.Shift
		fmt.Printf(
			"通知結果: status=%s / 講師: %s / 時間: %s〜%s / 場所: %s",
			sent.Status,
			shift.StaffName,
			shift.StartTime.Format("2006/01/02 15:04"),
			shift.EndTime.Format("15:04"),
			shift.Location,
		)
		if sent.ErrorMessage != "" {
			fmt.Printf(" / error: %s", sent.ErrorMessage)
		}
		fmt.Println()
	}

	return nil
}

func runNotifyLoop(ctx context.Context) error {
	interval := 5 * time.Minute
	if value := os.Getenv("SHIFT_NOTIFIER_NOTIFY_INTERVAL"); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("SHIFT_NOTIFIER_NOTIFY_INTERVAL is invalid: %w", err)
		}
		if parsed <= 0 {
			return fmt.Errorf("SHIFT_NOTIFIER_NOTIFY_INTERVAL must be positive")
		}
		interval = parsed
	}

	fmt.Printf("notification loop started: interval=%s\n", interval)
	for {
		if err := runNotifyOnce(ctx); err != nil {
			fmt.Printf("notification loop error: %s\n", err)
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
