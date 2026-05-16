package linebot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultAPIBaseURL     = "https://api.line.me"
	defaultDataAPIBaseURL = "https://api-data.line.me"
)

type Client struct {
	channelAccessToken string
	apiBaseURL         string
	dataAPIBaseURL     string
	httpClient         *http.Client
}

func NewClient(channelAccessToken string) *Client {
	return &Client{
		channelAccessToken: channelAccessToken,
		apiBaseURL:         defaultAPIBaseURL,
		dataAPIBaseURL:     defaultDataAPIBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) GetMessageContent(messageID string) (io.ReadCloser, error) {
	endpoint := fmt.Sprintf("%s/v2/bot/message/%s/content", c.dataAPIBaseURL, url.PathEscape(messageID))
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.authorize(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("LINE message content API failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}

func (c *Client) ReplyText(replyToken string, text string) error {
	payload := replyMessageRequest{
		ReplyToken: replyToken,
		Messages: []textMessage{
			{
				Type: "text",
				Text: text,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.apiBaseURL+"/v2/bot/message/reply", bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.authorize(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("LINE reply API failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) authorize(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.channelAccessToken)
}

type replyMessageRequest struct {
	ReplyToken string        `json:"replyToken"`
	Messages   []textMessage `json:"messages"`
}

type textMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
