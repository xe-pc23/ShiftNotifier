package linebot

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xe-pc23/shift-notifier/internal/parser"
	"github.com/xe-pc23/shift-notifier/internal/repository"
)

type ContentClient interface {
	GetMessageContent(ctx context.Context, messageID string) (io.ReadCloser, error)
	ReplyText(ctx context.Context, replyToken string, text string) error
}

type WebhookHandler struct {
	channelSecret string
	client        ContentClient
	store         *repository.Store
	importDir     string
}

func NewWebhookHandler(channelSecret string, client ContentClient, store *repository.Store, importDir string) *WebhookHandler {
	return &WebhookHandler{
		channelSecret: channelSecret,
		client:        client,
		store:         store,
		importDir:     importDir,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	if !ValidateSignature(h.channelSecret, body, r.Header.Get("X-Line-Signature")) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	for _, event := range payload.Events {
		if event.Type != "message" || event.Message.Type != "file" {
			continue
		}

		if err := h.handleFileMessage(r.Context(), event); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func ValidateSignature(channelSecret string, body []byte, signature string) bool {
	if channelSecret == "" || signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(channelSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (h *WebhookHandler) handleFileMessage(ctx context.Context, event webhookEvent) error {
	if !isExcelFile(event.Message.FileName) {
		return h.client.ReplyText(ctx, event.ReplyToken, "Excelファイル（.xlsx / .xls）を送信してください。")
	}

	savedPath, err := h.saveMessageContent(ctx, event.Message.ID, event.Message.FileName)
	if err != nil {
		_ = h.client.ReplyText(ctx, event.ReplyToken, "Excelファイルの保存に失敗しました。")
		return err
	}

	now := time.Now()
	shiftImport, err := h.store.BeginLineShiftImport(savedPath, now)
	if err != nil {
		_ = h.client.ReplyText(ctx, event.ReplyToken, "取込履歴の保存に失敗しました。")
		return err
	}

	shifts, err := parser.ParseExcel(savedPath, parser.DefaultSourceConfig)
	if err != nil {
		_ = h.store.MarkShiftImportFailed(shiftImport.ID, err.Error(), time.Now())
		_ = h.client.ReplyText(ctx, event.ReplyToken, fmt.Sprintf("Excelの解析に失敗しました。\n%s", err.Error()))
		return nil
	}

	savedShifts, err := h.store.UpsertShifts(shifts)
	if err != nil {
		_ = h.store.MarkShiftImportFailed(shiftImport.ID, err.Error(), time.Now())
		_ = h.client.ReplyText(ctx, event.ReplyToken, fmt.Sprintf("シフトの保存に失敗しました。\n%s", err.Error()))
		return nil
	}

	if err := h.store.MarkShiftImportImported(shiftImport.ID, time.Now()); err != nil {
		_ = h.client.ReplyText(ctx, event.ReplyToken, "取込完了後の履歴更新に失敗しました。")
		return err
	}

	return h.client.ReplyText(
		ctx,
		event.ReplyToken,
		fmt.Sprintf("Excelを取り込みました。\nファイル: %s\nシフト: %d件", event.Message.FileName, len(savedShifts)),
	)
}

func (h *WebhookHandler) saveMessageContent(ctx context.Context, messageID string, fileName string) (string, error) {
	content, err := h.client.GetMessageContent(ctx, messageID)
	if err != nil {
		return "", err
	}
	defer content.Close()

	if err := os.MkdirAll(h.importDir, 0755); err != nil {
		return "", err
	}

	safeName := sanitizeFileName(fileName)
	path := filepath.Join(h.importDir, fmt.Sprintf("%s_%s", messageID, safeName))
	tempFile, err := os.CreateTemp(h.importDir, ".download-*")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(tempFile, content); err != nil {
		_ = tempFile.Close()
		return "", err
	}

	if err := tempFile.Close(); err != nil {
		return "", err
	}

	if err := os.Rename(tempPath, path); err != nil {
		return "", err
	}

	return path, nil
}

func isExcelFile(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return ext == ".xlsx" || ext == ".xls"
}

func sanitizeFileName(fileName string) string {
	fileName = filepath.Base(fileName)
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	fileName = re.ReplaceAllString(fileName, "_")
	if strings.Trim(fileName, "._-") == "" {
		return "upload.xlsx"
	}
	return fileName
}

type webhookPayload struct {
	Events []webhookEvent `json:"events"`
}

type webhookEvent struct {
	Type       string         `json:"type"`
	ReplyToken string         `json:"replyToken"`
	Message    webhookMessage `json:"message"`
}

type webhookMessage struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
}
