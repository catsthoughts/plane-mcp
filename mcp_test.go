package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type MockServer struct {
	*Server
	mockResponses map[string]map[string]interface{}
}

func NewMockServer() *Server {
	return NewServer("test-workspace", "", "test-api-key", "http://localhost:8000")
}

func TestHandleInitialize(t *testing.T) {
	server := NewMockServer()
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := server.HandleRequest(context.Background(), req)

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", resp.JSONRPC)
	}
	if resp.ID != 1 {
		t.Errorf("Expected ID 1, got %v", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}

	result := resp.Result.(map[string]interface{})
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %v", result["protocolVersion"])
	}
	serverInfo := result["serverInfo"].(map[string]interface{})
	if serverInfo["name"] != "plane-mcp" {
		t.Errorf("Expected server name plane-mcp, got %v", serverInfo["name"])
	}
}

func TestHandleToolsList(t *testing.T) {
	server := NewMockServer()
	req := Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	resp := server.HandleRequest(context.Background(), req)

	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}

	result := resp.Result.(map[string]interface{})
	tools := result["tools"].([]map[string]interface{})

	expectedTools := []string{
		"get_defaults", "list_projects", "get_project", "list_issues", "get_issue",
		"create_issue", "update_issue", "list_states", "add_comment", "list_comments",
		"list_labels", "reopen_issue", "archive_issue", "list_members",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool["name"].(string)] = true
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool %s not found", name)
		}
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	server := NewMockServer()
	req := Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "unknown/method",
	}

	resp := server.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Errorf("Expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestHandleToolsCallUnknownTool(t *testing.T) {
	server := NewMockServer()
	params := json.RawMessage(`{"name":"nonexistent_tool","arguments":{}}`)
	req := Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  params,
	}

	resp := server.HandleRequest(context.Background(), req)

	if resp.Error == nil {
		t.Errorf("Expected error for unknown tool")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestTextResponse(t *testing.T) {
	server := NewMockServer()
	resp := server.textResponse("Test content", nil)

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", resp.JSONRPC)
	}

	result := resp.Result.(map[string]interface{})
	content := result["content"].([]map[string]interface{})
	if len(content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(content))
	}

	textContent := content[0]
	if textContent["text"] != "Test content" {
		t.Errorf("Expected 'Test content', got %v", textContent["text"])
	}
	if textContent["type"] != "text" {
		t.Errorf("Expected type 'text', got %v", textContent["type"])
	}
}

func TestErrorResponse(t *testing.T) {
	server := NewMockServer()
	resp := server.errorResponse(5, nil)

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", resp.JSONRPC)
	}
	if resp.ID != 5 {
		t.Errorf("Expected ID 5, got %v", resp.ID)
	}
}

func TestFormatProjects(t *testing.T) {
	projects := []Project{
		{ID: "1", Name: "Project 1", Identifier: "P1", Description: "Desc 1"},
		{ID: "2", Name: "Project 2", Identifier: "P2", Description: "Desc 2"},
	}

	output := formatProjects(projects)

	if output == "" {
		t.Errorf("Expected non-empty output")
	}
	if !strings.Contains(output, "Project 1") {
		t.Errorf("Expected Project 1 in output")
	}
	if !strings.Contains(output, "Project 2") {
		t.Errorf("Expected Project 2 in output")
	}
}

func TestFormatIssue(t *testing.T) {
	issue := Issue{
		ID:          "issue-123",
		Name:        "Test Issue",
		Description: "Test description",
		State:       "state-456",
		Priority:    "high",
		CreatedAt:   "2024-01-01T00:00:00Z",
	}

	output := formatIssue(issue)

	if output == "" {
		t.Errorf("Expected non-empty output")
	}
}

func TestFormatStates(t *testing.T) {
	states := []State{
		{ID: "1", Name: "Backlog", Group: "backlog"},
		{ID: "2", Name: "Todo", Group: "unstarted"},
		{ID: "3", Name: "Done", Group: "completed"},
	}

	output := formatStates(states)

	if output == "" {
		t.Errorf("Expected non-empty output")
	}
}

func TestFormatComment(t *testing.T) {
	comment := Comment{
		ID:          "comment-123",
		CommentHTML: "<p>Test comment</p>",
		CreatedAt:   "2024-01-01T00:00:00Z",
	}

	output := formatComment(comment)

	if output == "" {
		t.Errorf("Expected non-empty output")
	}
}

func TestFormatComments(t *testing.T) {
	comments := []Comment{
		{ID: "1", CommentHTML: "<p>Comment 1</p>", CreatedAt: "2024-01-01"},
		{ID: "2", CommentHTML: "<p>Comment 2</p>", CreatedAt: "2024-01-02"},
	}

	output := formatComments(comments)

	if output == "" {
		t.Errorf("Expected non-empty output")
	}
}

func TestFormatCommentsEmpty(t *testing.T) {
	comments := []Comment{}

	output := formatComments(comments)

	expected := "# Comments\n\nNo comments yet."
	if output != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, output)
	}
}