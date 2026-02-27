package notion

// API response types for Notion REST API

// DatabaseQueryResponse is the response from POST /v1/databases/{id}/query.
type DatabaseQueryResponse struct {
	Results    []PageObject `json:"results"`
	HasMore    bool         `json:"has_more"`
	NextCursor *string      `json:"next_cursor"`
}

// PageObject represents a Notion page object.
type PageObject struct {
	ID         string                         `json:"id"`
	Archived   bool                           `json:"archived"`
	Properties map[string]PropertyValueObject `json:"properties"`
}

// PropertyValueObject represents a property value in a Notion page.
type PropertyValueObject struct {
	Type     string        `json:"type"`
	RichText []RichTextObj `json:"rich_text,omitempty"`
	Title    []RichTextObj `json:"title,omitempty"`
}

// RichTextObj is a simplified Notion rich text object for reading property values.
type RichTextObj struct {
	PlainText string `json:"plain_text"`
}

// DatabaseObject represents a Notion database object.
type DatabaseObject struct {
	Properties map[string]DatabasePropertySchema `json:"properties"`
}

// DatabasePropertySchema represents a property schema in a database.
type DatabasePropertySchema struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// CreatePageRequest is the request body for POST /v1/pages.
type CreatePageRequest struct {
	Parent     Parent                 `json:"parent"`
	Properties map[string]interface{} `json:"properties"`
	Children   []interface{}          `json:"children,omitempty"`
}

// Parent represents the parent of a Notion page.
type Parent struct {
	DatabaseID string `json:"database_id"`
}

// AppendBlockChildrenRequest is the request body for PATCH /v1/blocks/{id}/children.
type AppendBlockChildrenRequest struct {
	Children []interface{} `json:"children"`
}

// ArchivePageRequest is the request body for PATCH /v1/pages/{id} to archive.
type ArchivePageRequest struct {
	Archived bool `json:"archived"`
}

// CreatePageResponse is the response from POST /v1/pages.
type CreatePageResponse struct {
	ID string `json:"id"`
}

// FileUploadCreateRequest is the request body for POST /v1/file_uploads.
type FileUploadCreateRequest struct {
	Mode string `json:"mode"`
}

// FileUploadCreateResponse is the response from POST /v1/file_uploads.
type FileUploadCreateResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// NotionErrorResponse represents an error from the Notion API.
type NotionErrorResponse struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
