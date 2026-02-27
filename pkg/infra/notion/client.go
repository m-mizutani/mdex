package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/mdex/pkg/domain"
	"github.com/m-mizutani/mdex/pkg/utils/dryrun"
	"github.com/m-mizutani/mdex/pkg/utils/logging"
	"github.com/m-mizutani/mdex/pkg/utils/safe"
)

const (
	baseURL       = "https://api.notion.com/v1"
	notionVersion = "2022-06-28"
	maxChildren   = 100
)

// Client implements domain.NotionClient using net/http.
type Client struct {
	token      string
	httpClient *http.Client
}

// New creates a new Notion API client.
func New(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Ensure Client implements domain.NotionClient.
var _ domain.NotionClient = (*Client)(nil)

// QueryDatabase retrieves all records from the specified Notion database.
// If domainValue is non-empty, only pages matching the domain filter are returned.
func (c *Client) QueryDatabase(ctx context.Context, databaseID string, pathProperty string, hashProperty string, domainProperty string, domainValue string) ([]domain.NotionPage, error) {
	if dryrun.IsDryRun(ctx) {
		logging.From(ctx).Info("dry-run: would query database", "database_id", databaseID)
		return nil, nil
	}

	var allPages []domain.NotionPage
	var startCursor *string

	for {
		body := map[string]interface{}{
			"page_size": 100,
		}
		if domainValue != "" {
			body["filter"] = map[string]interface{}{
				"property": domainProperty,
				"select": map[string]interface{}{
					"equals": domainValue,
				},
			}
		}
		if startCursor != nil {
			body["start_cursor"] = *startCursor
		}

		resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/databases/%s/query", databaseID), body)
		if err != nil {
			return nil, goerr.Wrap(err, "querying database", goerr.V("databaseID", databaseID))
		}
		defer safe.Close(ctx, resp.Body)

		var result DatabaseQueryResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, goerr.Wrap(err, "decoding database query response")
		}

		for _, page := range result.Results {
			if page.Archived {
				continue
			}

			np := domain.NotionPage{ID: page.ID}

			if prop, ok := page.Properties[pathProperty]; ok {
				np.MdexPath = extractPlainText(prop)
			}
			if prop, ok := page.Properties[hashProperty]; ok {
				np.MdexHash = extractPlainText(prop)
			}

			allPages = append(allPages, np)
		}

		if !result.HasMore || result.NextCursor == nil {
			break
		}
		startCursor = result.NextCursor
	}

	return allPages, nil
}

// GetTitleProperty retrieves the name of the title property from the database schema.
func (c *Client) GetTitleProperty(ctx context.Context, databaseID string) (string, error) {
	if dryrun.IsDryRun(ctx) {
		logging.From(ctx).Info("dry-run: would get database schema", "database_id", databaseID)
		return "Name", nil
	}

	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/databases/%s", databaseID), nil)
	if err != nil {
		return "", goerr.Wrap(err, "getting database schema", goerr.V("databaseID", databaseID))
	}
	defer safe.Close(ctx, resp.Body)

	var db DatabaseObject
	if err := json.NewDecoder(resp.Body).Decode(&db); err != nil {
		return "", goerr.Wrap(err, "decoding database schema")
	}

	for name, prop := range db.Properties {
		if prop.Type == "title" {
			return name, nil
		}
	}

	return "", goerr.New("no title property found in database", goerr.V("databaseID", databaseID))
}

// CreatePage creates a new page in the specified database.
// If blocks exceed 100, only the first 100 are included in the create request,
// and the rest are appended via AppendBlocks.
func (c *Client) CreatePage(ctx context.Context, databaseID string, properties map[string]interface{}, blocks []interface{}) (string, error) {
	if dryrun.IsDryRun(ctx) {
		logging.From(ctx).Info("dry-run: would create page", "database_id", databaseID, "block_count", len(blocks))
		return "dry-run-page-id", nil
	}

	// Split blocks: first batch in create, rest appended
	var firstBatch []interface{}
	var remaining []interface{}
	if len(blocks) > maxChildren {
		firstBatch = blocks[:maxChildren]
		remaining = blocks[maxChildren:]
	} else {
		firstBatch = blocks
	}

	req := CreatePageRequest{
		Parent:     Parent{DatabaseID: databaseID},
		Properties: properties,
		Children:   firstBatch,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/pages", req)
	if err != nil {
		return "", goerr.Wrap(err, "creating page", goerr.V("databaseID", databaseID))
	}
	defer safe.Close(ctx, resp.Body)

	var result CreatePageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", goerr.Wrap(err, "decoding create page response")
	}

	// Append remaining blocks in batches
	for len(remaining) > 0 {
		batch := remaining
		if len(batch) > maxChildren {
			batch = remaining[:maxChildren]
			remaining = remaining[maxChildren:]
		} else {
			remaining = nil
		}

		if err := c.AppendBlocks(ctx, result.ID, batch); err != nil {
			return result.ID, goerr.Wrap(err, "appending blocks", goerr.V("pageID", result.ID))
		}
	}

	return result.ID, nil
}

// AppendBlocks appends blocks to an existing page.
func (c *Client) AppendBlocks(ctx context.Context, pageID string, blocks []interface{}) error {
	if dryrun.IsDryRun(ctx) {
		logging.From(ctx).Info("dry-run: would append blocks", "page_id", pageID, "block_count", len(blocks))
		return nil
	}

	req := AppendBlockChildrenRequest{
		Children: blocks,
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/blocks/%s/children", pageID), req)
	if err != nil {
		return goerr.Wrap(err, "appending blocks", goerr.V("pageID", pageID))
	}
	defer safe.Close(ctx, resp.Body)

	return nil
}

// GetPage retrieves a page by its ID. This is primarily used for testing.
func (c *Client) GetPage(ctx context.Context, pageID string) (*PageObject, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/pages/%s", pageID), nil)
	if err != nil {
		return nil, goerr.Wrap(err, "getting page", goerr.V("pageID", pageID))
	}
	defer safe.Close(ctx, resp.Body)

	var page PageObject
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, goerr.Wrap(err, "decoding page response")
	}

	return &page, nil
}

// ArchivePage archives (soft-deletes) a Notion page.
func (c *Client) ArchivePage(ctx context.Context, pageID string) error {
	if dryrun.IsDryRun(ctx) {
		logging.From(ctx).Info("dry-run: would archive page", "page_id", pageID)
		return nil
	}

	req := ArchivePageRequest{Archived: true}

	resp, err := c.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/pages/%s", pageID), req)
	if err != nil {
		return goerr.Wrap(err, "archiving page", goerr.V("pageID", pageID))
	}
	defer safe.Close(ctx, resp.Body)

	return nil
}

// UploadFile uploads a file to Notion via the File Upload API (two-step process).
func (c *Client) UploadFile(ctx context.Context, filename string, contentType string, body io.Reader) (string, error) {
	if dryrun.IsDryRun(ctx) {
		logging.From(ctx).Info("dry-run: would upload file", "filename", filename)
		return "dry-run-file-upload-id", nil
	}

	// Step 1: Create file upload
	createReq := FileUploadCreateRequest{Mode: "single_part"}
	resp, err := c.doRequest(ctx, http.MethodPost, "/file_uploads", createReq)
	if err != nil {
		return "", goerr.Wrap(err, "creating file upload", goerr.V("filename", filename))
	}
	defer safe.Close(ctx, resp.Body)

	var createResp FileUploadCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return "", goerr.Wrap(err, "decoding file upload create response")
	}

	// Step 2: Send file content
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	partHeader.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return "", goerr.Wrap(err, "creating form file")
	}
	if _, err := io.Copy(part, body); err != nil {
		return "", goerr.Wrap(err, "copying file content")
	}
	if err := writer.Close(); err != nil {
		return "", goerr.Wrap(err, "closing multipart writer")
	}

	sendReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/file_uploads/%s/send", baseURL, createResp.ID), bytes.NewReader(buf.Bytes()))
	if err != nil {
		return "", goerr.Wrap(err, "creating send request")
	}
	sendReq.Header.Set("Authorization", "Bearer "+c.token)
	sendReq.Header.Set("Notion-Version", notionVersion)
	sendReq.Header.Set("Content-Type", writer.FormDataContentType())

	sendResp, err := c.doHTTPWithRetry(ctx, sendReq)
	if err != nil {
		return "", goerr.Wrap(err, "sending file", goerr.V("fileUploadID", createResp.ID))
	}
	defer safe.Close(ctx, sendResp.Body)

	if sendResp.StatusCode < 200 || sendResp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(sendResp.Body)
		return "", goerr.New("file upload send failed",
			goerr.V("status", sendResp.StatusCode),
			goerr.V("body", string(respBody)),
		)
	}

	return createResp.ID, nil
}

// doRequest performs an HTTP request to the Notion API with automatic retry on 429.
func (c *Client) doRequest(ctx context.Context, method string, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, goerr.Wrap(err, "marshaling request body")
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, goerr.Wrap(err, "creating request")
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", notionVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.doHTTPWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		safe.Close(ctx, resp.Body)
		return nil, goerr.New("notion API error",
			goerr.V("status", resp.StatusCode),
			goerr.V("method", method),
			goerr.V("path", path),
			goerr.V("body", string(respBody)),
		)
	}

	return resp, nil
}

// doHTTPWithRetry performs an HTTP request with retry on 429 status.
func (c *Client) doHTTPWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	for {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, goerr.Wrap(err, "executing HTTP request")
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		safe.Close(ctx, resp.Body)

		retryAfter := 1 // default 1 second
		if val := resp.Header.Get("Retry-After"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil {
				retryAfter = parsed
			}
		}

		logging.From(ctx).Warn("rate limited, waiting before retry",
			"retry_after_seconds", retryAfter,
			"url", req.URL.String(),
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(retryAfter) * time.Second):
		}

		// Re-create the body if it was consumed
		if req.GetBody != nil {
			newBody, err := req.GetBody()
			if err != nil {
				return nil, goerr.Wrap(err, "re-creating request body for retry")
			}
			req.Body = newBody
		}
	}
}

func extractPlainText(prop PropertyValueObject) string {
	var parts []string
	switch prop.Type {
	case "rich_text":
		for _, rt := range prop.RichText {
			parts = append(parts, rt.PlainText)
		}
	case "title":
		for _, rt := range prop.Title {
			parts = append(parts, rt.PlainText)
		}
	}
	return strings.Join(parts, "")
}
