package model

import "time"

type Shift struct {
	StaffName string //"講師名",
	StartTime time.Time //ex)2026/04/30 19:00,
	EndTime   time.Time //ex)2026/04/30 21:00,
	Location  string //"教室名"
}
