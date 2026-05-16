package model

import "time"

type ShiftImportStatus string

const (
	ShiftImportStatusParsing  ShiftImportStatus = "parsing"
	ShiftImportStatusImported ShiftImportStatus = "imported"
	ShiftImportStatusFailed   ShiftImportStatus = "failed"
)

type ShiftImport struct {
	ID           uint
	SourceType   string
	FileName     string
	FilePath     string
	FileHash     string
	ImportedAt   *time.Time
	Status       ShiftImportStatus
	ErrorMessage string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
