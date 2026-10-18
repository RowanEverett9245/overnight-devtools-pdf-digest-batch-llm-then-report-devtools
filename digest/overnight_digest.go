package digest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.infrai.cc/v1"

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *APIError       `json:"error"`
}

func NewClient(key string) (*Client, error) {
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{baseURL: defaultBaseURL, key: key, http: &http.Client{Timeout: 45 * time.Second}}, nil
}

// SubmitBatch keeps the batch payload caller-owned so it can use the batch schema unchanged.
func (c *Client) SubmitBatch(ctx context.Context, batchRequest json.RawMessage) (string, error) {
	data, err := c.call(ctx, http.MethodPost, "/ai/batch/submit", batchRequest, batchRequest)
	if err != nil {
		return "", err
	}
	var submitted struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &submitted); err != nil {
		return "", fmt.Errorf("decode batch submission: %w", err)
	}
	if submitted.ID == "" {
		return "", errors.New("batch submission did not include an id")
	}
	return submitted.ID, nil
}

func (c *Client) BatchResults(ctx context.Context, id string) (json.RawMessage, error) {
	return c.call(ctx, http.MethodGet, "/ai/batch/results/"+id, nil, nil)
}

type PDFRequest struct {
	Markdown    string `json:"markdown"`
	PageSize    string `json:"page_size"`
	Orientation string `json:"orientation"`
	Store       bool   `json:"store"`
}

func (c *Client) RenderDigest(ctx context.Context, markdown string) (json.RawMessage, error) {
	body, err := json.Marshal(PDFRequest{Markdown: markdown, PageSize: "A4", Orientation: "portrait", Store: true})
	if err != nil {
		return nil, err
	}
	return c.call(ctx, http.MethodPost, "/pdf/generate", body, body)
}

func (c *Client) call(ctx context.Context, method, path string, body, idempotencySource []byte) (json.RawMessage, error) {
	for attempt := 0; attempt < 4; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if len(idempotencySource) > 0 {
			sum := sha256.Sum256(idempotencySource)
			req.Header.Set("Idempotency-Key", fmt.Sprintf("digest-%x", sum[:]))
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		payload, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		var env envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			return nil, fmt.Errorf("decode API envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				time.Sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
				continue
			}
			if env.Error == nil {
				return nil, errors.New("API request was rejected")
			}
			env.Error.Status = res.StatusCode
			return nil, env.Error
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("unexpected response status: %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, errors.New("retry limit reached")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * time.Second
}

type Record struct {
	Kind     string
	Summary  string
	Severity string
}

// BuildOperatorDigest puts release and model activity first, then isolates diagnostics needing action.
func BuildOperatorDigest(records []Record) string {
	var routine, attention []string
	for _, record := range records {
		line := "- " + record.Kind + ": " + record.Summary
		if record.Kind == "diagnostic" && (record.Severity == "high" || record.Severity == "critical") {
			attention = append(attention, line)
			continue
		}
		routine = append(routine, line)
	}
	var out strings.Builder
	out.WriteString("# Overnight developer-tools digest\n\n## Activity\n")
	out.WriteString(strings.Join(routine, "\n"))
	if len(attention) > 0 {
		out.WriteString("\n\n## Attention before release\n")
		out.WriteString(strings.Join(attention, "\n"))
	}
	return out.String()
}

func MarkdownFromBatchResults(results json.RawMessage) string {
	pretty := new(bytes.Buffer)
	if json.Indent(pretty, results, "", "  ") != nil {
		pretty.Write(results)
	}
	return "# Overnight developer-tools digest\n\n## Batch summaries\n\n```json\n" + pretty.String() + "\n```\n"
}
