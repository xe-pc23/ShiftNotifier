package linebot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSignature(t *testing.T) {
	secret := "channel-secret"
	body := []byte(`{"events":[]}`)

	if !ValidateSignature(secret, body, testSignature(secret, body)) {
		t.Fatal("ValidateSignature() = false, want true")
	}

	if ValidateSignature(secret, body, "invalid") {
		t.Fatal("ValidateSignature() = true, want false")
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	handler := NewWebhookHandler("secret", &fakeLINEClient{}, nil, t.TempDir())
	req := httptest.NewRequest(http.MethodPost, "/webhook/line", strings.NewReader(`{"events":[]}`))
	req.Header.Set("X-Line-Signature", "invalid")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestWebhookRejectsNonExcelFile(t *testing.T) {
	secret := "secret"
	body := []byte(`{"events":[{"type":"message","replyToken":"reply-token","message":{"type":"file","id":"message-id","fileName":"memo.txt","fileSize":10}}]}`)
	client := &fakeLINEClient{}
	handler := NewWebhookHandler(secret, client, nil, t.TempDir())
	req := signedRequest(secret, body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(client.replyText, "Excelファイル") {
		t.Fatalf("replyText = %q, want Excel warning", client.replyText)
	}
}

func TestWebhookRepliesUserID(t *testing.T) {
	secret := "secret"
	body := []byte(`{"events":[{"type":"message","replyToken":"reply-token","source":{"type":"user","userId":"U123456"},"message":{"type":"text","id":"message-id","text":"id"}}]}`)
	client := &fakeLINEClient{}
	handler := NewWebhookHandler(secret, client, nil, t.TempDir())
	req := signedRequest(secret, body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(client.replyText, "U123456") {
		t.Fatalf("replyText = %q, want user ID", client.replyText)
	}
}

func TestSanitizeFileName(t *testing.T) {
	got := sanitizeFileName(filepath.Join("..", "危険な ファイル.xlsx"))
	if got != "_.xlsx" {
		t.Fatalf("sanitizeFileName() = %q, want %q", got, "_.xlsx")
	}
}

func TestSaveMessageContentUsesFinalPath(t *testing.T) {
	importDir := t.TempDir()
	client := &fakeLINEClient{content: []byte("excel bytes")}
	handler := NewWebhookHandler("secret", client, nil, importDir)

	path, err := handler.saveMessageContent(context.Background(), "message-id", "shift.xlsx")
	if err != nil {
		t.Fatal(err)
	}

	if filepath.Base(path) != "message-id_shift.xlsx" {
		t.Fatalf("saved file = %q, want %q", filepath.Base(path), "message-id_shift.xlsx")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "excel bytes" {
		t.Fatalf("content = %q, want %q", string(content), "excel bytes")
	}

	entries, err := os.ReadDir(importDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
}

type fakeLINEClient struct {
	content   []byte
	replyText string
}

func (c *fakeLINEClient) GetMessageContent(ctx context.Context, messageID string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(c.content)), nil
}

func (c *fakeLINEClient) ReplyText(ctx context.Context, replyToken string, text string) error {
	c.replyText = text
	return nil
}

func signedRequest(secret string, body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook/line", bytes.NewReader(body))
	req.Header.Set("X-Line-Signature", testSignature(secret, body))
	return req
}

func testSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
