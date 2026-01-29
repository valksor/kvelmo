package wrike

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/valksor/go-mehrhof/internal/provider/httpclient"
	"github.com/valksor/go-mehrhof/internal/provider/token"
)

const (
	defaultBaseURL = "https://www.wrike.com/api/v4"
)

// Config holds client configuration.
type Config struct {
	Token string
	Host  string // Optional: override default API base URL
}

// Client wraps the Wrike API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	folderID   string // Default folder for list/create operations
	spaceID    string // Default space for list operations
}

// NewClient creates a new Wrike API client.
func NewClient(token, host string) *Client {
	baseURL := defaultBaseURL
	if host != "" {
		baseURL = strings.TrimSuffix(host, "/")
	}

	return &Client{
		httpClient: httpclient.NewHTTPClient(),
		baseURL:    baseURL,
		token:      token,
	}
}

// NewClientWithConfig creates a new Wrike API client with full configuration.
func NewClientWithConfig(cfg ClientConfig) *Client {
	baseURL := defaultBaseURL
	if cfg.Host != "" {
		baseURL = strings.TrimSuffix(cfg.Host, "/")
	}

	return &Client{
		httpClient: httpclient.NewHTTPClient(),
		baseURL:    baseURL,
		token:      cfg.Token,
		folderID:   cfg.FolderID,
		spaceID:    cfg.SpaceID,
	}
}

// ClientConfig holds extended client configuration.
type ClientConfig struct {
	Token    string
	Host     string
	FolderID string
	SpaceID  string
}

// ResolveToken resolves the Wrike API token.
// The configToken should be from config.yaml and may use ${VAR} syntax.
func ResolveToken(configToken string) (string, error) {
	return token.ResolveToken(token.Config("WRIKE", configToken))
}

// doRequest performs an HTTP request to the Wrike API.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, result any) error {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return wrapAPIError(err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return wrapAPIError(httpclient.NewHTTPError(resp.StatusCode, string(respBody)))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// doRequestWithRetry performs an HTTP request with exponential backoff retry.
// Retries on rate limit errors (429) and service unavailable (503).
func (c *Client) doRequestWithRetry(ctx context.Context, method, path string, body io.Reader, result any) error {
	return httpclient.WithRetry(ctx, httpclient.DefaultRetryConfig(), func() error {
		return c.doRequest(ctx, method, path, body, result)
	})
}

// GetTask fetches a task by ID.
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	var response taskResponse
	if err := c.doRequestWithRetry(ctx, http.MethodGet, "/tasks/"+url.PathEscape(taskID), nil, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, ErrTaskNotFound
	}

	return &response.Data[0], nil
}

// GetTaskByPermalink fetches a task by permalink URL
// Extracts the numeric ID from the permalink and uses the standard task endpoint.
func (c *Client) GetTaskByPermalink(ctx context.Context, permalink string) (*Task, error) {
	numericID := ExtractNumericID(permalink)
	if numericID == "" {
		return nil, fmt.Errorf("%w: invalid permalink format", ErrInvalidReference)
	}

	// Use the extracted numeric ID directly with standard endpoint
	// The Wrike API v4 doesn't document a ?permalink= query parameter,
	// so we extract the numeric ID and use the standard /tasks/{id} endpoint
	return c.GetTask(ctx, numericID)
}

// GetTaskByPermalinkParam fetches a task by permalink URL using query parameter.
// Uses GET /tasks?permalink=... which is the official Wrike API method.
func (c *Client) GetTaskByPermalinkParam(ctx context.Context, permalink string) (*Task, error) {
	// Build query string with permalink parameter
	path := "/tasks?permalink=" + url.QueryEscape(permalink)

	var response taskResponse
	if err := c.doRequestWithRetry(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, ErrTaskNotFound
	}

	return &response.Data[0], nil
}

// GetFolderByPermalink resolves a numeric folder/project ID to its API ID.
// Uses GET /folders?permalink=https://www.wrike.com/open.htm?id={numericID}
// Works for folders, projects, and subfolders.
func (c *Client) GetFolderByPermalink(ctx context.Context, numericID string) (*Folder, error) {
	permalink := BuildPermalinkURL(numericID)
	path := "/folders?permalink=" + url.QueryEscape(permalink)

	var response folderResponse
	if err := c.doRequestWithRetry(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, ErrFolderNotFound
	}

	return &response.Data[0], nil
}

// GetTasks fetches multiple tasks by IDs.
func (c *Client) GetTasks(ctx context.Context, taskIDs []string) ([]Task, error) {
	var response taskResponse
	if err := c.doRequestWithRetry(ctx, http.MethodGet, "/tasks/"+url.PathEscape(strings.Join(taskIDs, ",")), nil, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

// GetTasksInFolder fetches all tasks in a folder.
func (c *Client) GetTasksInFolder(ctx context.Context, folderID string) ([]Task, error) {
	var response taskResponse
	if err := c.doRequestWithRetry(ctx, http.MethodGet, "/folders/"+url.PathEscape(folderID)+"/tasks", nil, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

// GetTasksInSpace fetches all tasks in a space.
func (c *Client) GetTasksInSpace(ctx context.Context, spaceID string) ([]Task, error) {
	var response taskResponse
	if err := c.doRequestWithRetry(ctx, http.MethodGet, "/spaces/"+url.PathEscape(spaceID)+"/tasks", nil, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

// GetComments fetches comments for a task with pagination support.
func (c *Client) GetComments(ctx context.Context, taskID string) ([]Comment, error) {
	var allComments []Comment
	path := "/tasks/" + url.PathEscape(taskID) + "/comments"

	for {
		var response commentsResponse
		if err := c.doRequestWithRetry(ctx, http.MethodGet, path, nil, &response); err != nil {
			if len(allComments) == 0 {
				return nil, err
			}

			break // Return what we have on error after first page
		}

		allComments = append(allComments, response.Data...)

		// Check if there's a next page
		if response.NextPage == "" {
			break
		}
		path = response.NextPage
	}

	return allComments, nil
}

// GetAttachments fetches attachments for a task.
func (c *Client) GetAttachments(ctx context.Context, taskID string) ([]Attachment, error) {
	var response attachmentsResponse
	if err := c.doRequestWithRetry(ctx, http.MethodGet, "/tasks/"+url.PathEscape(taskID)+"/attachments", nil, &response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

// DownloadAttachment downloads an attachment by ID.
func (c *Client) DownloadAttachment(ctx context.Context, attachmentID string) (io.ReadCloser, string, error) {
	reqURL := c.baseURL + "/attachments/" + url.PathEscape(attachmentID) + "/download"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", wrapAPIError(err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()

		return nil, "", wrapAPIError(httpclient.NewHTTPError(resp.StatusCode, "download failed"))
	}

	return resp.Body, resp.Header.Get("Content-Disposition"), nil
}

// PostComment adds a comment to a task.
func (c *Client) PostComment(ctx context.Context, taskID, text string) (*Comment, error) {
	requestBody := map[string]string{
		"text": text,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	var response commentResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPost, "/tasks/"+url.PathEscape(taskID)+"/comments",
		strings.NewReader(string(bodyBytes)), &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, errors.New("no comment returned")
	}

	return &response.Data[0], nil
}

// UpdateTaskStatus updates the status of a task.
func (c *Client) UpdateTaskStatus(ctx context.Context, taskID, status string) error {
	requestBody := map[string]string{
		"status": status,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	var response taskResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPut, "/tasks/"+url.PathEscape(taskID),
		strings.NewReader(string(bodyBytes)), &response); err != nil {
		return err
	}

	return nil
}

// UpdateTaskTags updates the tags of a task.
func (c *Client) UpdateTaskTags(ctx context.Context, taskID string, tags []string) error {
	requestBody := map[string]any{
		"tags": tags,
	}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	var response taskResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPut, "/tasks/"+url.PathEscape(taskID),
		strings.NewReader(string(bodyBytes)), &response); err != nil {
		return err
	}

	return nil
}

// CreateTaskOptions holds options for creating a new task.
type CreateTaskOptions struct {
	Title         string
	Description   string
	Priority      string
	Status        string
	DependencyIDs []string // Task IDs this task depends on (predecessors)
}

// CreateTask creates a new task in a folder.
// If DependencyIDs are specified, dependencies are created after task creation.
func (c *Client) CreateTask(ctx context.Context, folderID string, opts CreateTaskOptions) (*Task, error) {
	requestBody := map[string]any{
		"title": opts.Title,
	}
	if opts.Description != "" {
		requestBody["description"] = opts.Description
	}
	if opts.Priority != "" {
		requestBody["priority"] = opts.Priority
	}
	if opts.Status != "" {
		requestBody["status"] = opts.Status
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	var response taskResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPost, "/folders/"+url.PathEscape(folderID)+"/tasks",
		strings.NewReader(string(bodyBytes)), &response); err != nil {
		return nil, err
	}

	if len(response.Data) == 0 {
		return nil, errors.New("no task returned")
	}

	task := &response.Data[0]

	// Create dependencies if specified
	// In Wrike, a dependency means: predecessorId must complete before successorId can start
	// So if task A depends on task B, B is the predecessor and A (the new task) is the successor
	if len(opts.DependencyIDs) > 0 {
		for _, predecessorID := range opts.DependencyIDs {
			if err := c.CreateDependency(ctx, predecessorID, task.ID); err != nil {
				// Log error but don't fail the entire task creation
				// Dependencies can be added later if needed
				continue
			}
		}
		task.DependencyIDs = opts.DependencyIDs
	}

	return task, nil
}

// Dependency represents a Wrike task dependency.
type Dependency struct {
	ID            string `json:"id"`
	PredecessorID string `json:"predecessorId"`
	SuccessorID   string `json:"successorId"`
	RelationType  string `json:"relationType"` // FinishToStart, StartToStart, etc.
}

// dependencyResponse wraps the Wrike API response for dependencies.
type dependencyResponse struct {
	Data []Dependency `json:"data"`
}

// CreateDependency creates a dependency between two tasks.
// The predecessor must complete before the successor can start.
func (c *Client) CreateDependency(ctx context.Context, predecessorID, successorID string) error {
	requestBody := map[string]any{
		"predecessorId": predecessorID,
		"successorId":   successorID,
		"relationType":  "FinishToStart", // Most common dependency type
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	var response dependencyResponse
	if err := c.doRequestWithRetry(ctx, http.MethodPost, "/dependencies",
		strings.NewReader(string(bodyBytes)), &response); err != nil {
		return fmt.Errorf("create dependency: %w", err)
	}

	return nil
}

// GetTaskDependencies returns the dependencies for a task.
func (c *Client) GetTaskDependencies(ctx context.Context, taskID string) ([]Dependency, error) {
	var response dependencyResponse
	if err := c.doRequestWithRetry(ctx, http.MethodGet,
		"/tasks/"+url.PathEscape(taskID)+"/dependencies", nil, &response); err != nil {
		return nil, fmt.Errorf("get dependencies: %w", err)
	}

	return response.Data, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Wrike API Types
// ──────────────────────────────────────────────────────────────────────────────

// Task represents a Wrike task.
type Task struct {
	CreatedDate   time.Time `json:"createdDate"`
	UpdatedDate   time.Time `json:"updatedDate"`
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Status        string    `json:"status"`
	Priority      string    `json:"priority"`
	Permalink     string    `json:"permalink"`
	SubTaskIDs    []string  `json:"subTaskIds"`
	Tags          []string  `json:"tags"`
	DependencyIDs []string  `json:"dependencyIds,omitempty"` // Task IDs this task depends on
}

// Comment represents a Wrike comment.
type Comment struct {
	CreatedDate time.Time `json:"createdDate"`
	UpdatedDate time.Time `json:"updatedDate"`
	ID          string    `json:"id"`
	Text        string    `json:"text"`
	AuthorID    string    `json:"authorId"`
	AuthorName  string    `json:"authorName,omitempty"`
}

// Attachment represents a Wrike attachment.
type Attachment struct {
	CreatedDate time.Time `json:"createdDate"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size,omitempty"`
}

// Response wrappers for Wrike API.
type taskResponse struct {
	Data []Task `json:"data"`
}

type commentsResponse struct {
	NextPage string    `json:"nextPage,omitempty"`
	Data     []Comment `json:"data"`
}

type commentResponse struct {
	Data []Comment `json:"data"`
}

type attachmentsResponse struct {
	Data []Attachment `json:"data"`
}

// Folder represents a Wrike folder or project.
// Projects are folders with additional properties (Project field is non-nil).
type Folder struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	ChildIDs  []string       `json:"childIds"`
	Scope     string         `json:"scope"` // "WsFolder", "WsProject", "RbFolder"
	Permalink string         `json:"permalink"`
	Project   *FolderProject `json:"project,omitempty"` // Non-nil if this is a project
}

// FolderProject contains project-specific properties (owners, dates, status).
// This is embedded in Folder when the folder is a project.
type FolderProject struct {
	AuthorID    string    `json:"authorId"`
	OwnerIDs    []string  `json:"ownerIds"`
	Status      string    `json:"status"` // "Green", "Yellow", "Red", "Completed", "OnHold", "Cancelled"
	CreatedDate time.Time `json:"createdDate"`
	StartDate   string    `json:"startDate,omitempty"`
	EndDate     string    `json:"endDate,omitempty"`
}

type folderResponse struct {
	Data []Folder `json:"data"`
}
