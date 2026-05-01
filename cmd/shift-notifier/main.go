package main

import (
	"fmt"
	"log"
	"os"

	"github.com/xe-pc23/shift-notifier/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
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

	for _, shift := range shifts {
		fmt.Printf(
			"講師: %s / 時間: %s〜%s / 場所: %s\n",
			shift.StaffName,
			shift.StartTime.Format("2006/01/02 15:04"),
			shift.EndTime.Format("15:04"),
			shift.Location,
		)
	}
}
