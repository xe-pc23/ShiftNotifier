package model

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"time"
)

type Shift struct {
	ID        uint
	StaffName string    //"講師名",
	StartTime time.Time //ex)2026/04/30 19:00,
	EndTime   time.Time //ex)2026/04/30 21:00,
	Location  string    //"教室名"

	SourceKey   string
	ContentHash string
}

func ShiftSourceKey(shift Shift) string {
	if shift.SourceKey != "" {
		return shift.SourceKey
	}

	key := fmt.Sprintf(
		"%s|%s|%s",
		shift.StaffName,
		shift.StartTime.Format(time.RFC3339),
		shift.Location,
	)

	return hashString(key)
}

func ShiftContentHash(shift Shift) string {
	if shift.ContentHash != "" {
		return shift.ContentHash
	}

	key := fmt.Sprintf(
		"%s|%s|%s|%s",
		shift.StaffName,
		shift.StartTime.Format(time.RFC3339),
		shift.EndTime.Format(time.RFC3339),
		shift.Location,
	)

	return hashString(key)
}

func hashString(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])
}
