package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type Server struct {
	tools           []Tool
	workspace       string
	defaultProject  string
	apiKey          string
	baseURL         string
}

func NewServer(workspace, defaultProject, apiKey, baseURL string) *Server {
	baseURL = strings.TrimRight(baseURL, "/")
	s := &Server{
		workspace:      workspace,
		defaultProject: defaultProject,
		apiKey:         apiKey,
		baseURL:        baseURL,
		tools:          []Tool{},
	}
	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	s.tools = []Tool{
		{
			Name:        "get_defaults",
			Description: "Get default workspace and project configured via environment variables",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "list_projects",
			Description: "List all projects in a workspace",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug"}
				}
			}`),
		},
		{
			Name:        "get_project",
			Description: "Get a specific project by ID",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID. Use list_projects first to get project IDs."}
				}
			}`),
		},
		{
			Name:        "list_issues",
			Description: "List issues in a workspace",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID. Use list_projects first to get project IDs."},
					"search": {"type": "string", "description": "Search issues by name or description."},
					"state": {"type": "string"}
				}
			}`),
		},
		{
			Name:        "get_issue",
			Description: "Get a specific issue by ID",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id"],
				"properties": {
					"workspace": {"type": "string"},
					"issue_id": {"type": "string"}
				}
			}`),
		},
		{
			Name:        "create_issue",
			Description: "Create a new issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id", "name"],
				"properties": {
					"workspace": {"type": "string"},
					"project_id": {"type": "string"},
					"name": {"type": "string"},
					"description": {"type": "string"},
					"state": {"type": "string"},
					"priority": {"type": "string"},
					"assignees": {"type": "array", "items": {"type": "string"}, "description": "List of member UUIDs. Use list_members first to get valid UUIDs."},
					"labels": {"type": "array", "items": {"type": "string"}, "description": "List of label UUIDs. Use list_labels first to get valid UUIDs."}
				}
			}`),
		},
		{
			Name:        "update_issue",
			Description: "Update an existing issue (name, description, state, priority, assignees, labels, parent, start_date, target_date)",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id"],
				"properties": {
					"workspace": {"type": "string"},
					"issue_id": {"type": "string"},
					"name": {"type": "string"},
					"description": {"type": "string"},
					"state": {"type": "string"},
					"priority": {"type": "string", "enum": ["urgent", "high", "medium", "low", "none"]},
					"assignees": {"type": "array", "items": {"type": "string"}, "description": "List of member UUIDs. Use list_members first to get valid UUIDs."},
					"labels": {"type": "array", "items": {"type": "string"}, "description": "List of label UUIDs. Use list_labels first to get valid UUIDs."},
					"parent": {"type": "string"},
					"start_date": {"type": "string", "description": "YYYY-MM-DD format"},
					"target_date": {"type": "string", "description": "YYYY-MM-DD format"}
				}
			}`),
		},
		{
			Name:        "list_states",
			Description: "List all states in a project",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID (not identifier). Use list_projects first to get project IDs."}
				}
			}`),
		},
		{
			Name:        "add_comment",
			Description: "Add a comment to an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "comment"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID"},
					"comment": {"type": "string", "description": "Comment text"}
				}
			}`),
		},
		{
			Name:        "add_link",
			Description: "Add a link to an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "url"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-5 format"},
					"url": {"type": "string", "description": "URL to add"},
					"title": {"type": "string", "description": "Link title (optional)"}
				}
			}`),
		},
		{
			Name:        "update_link",
			Description: "Update a link on an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "link_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-5 format"},
					"link_id": {"type": "string", "description": "Link UUID to update"},
					"url": {"type": "string", "description": "New URL (optional)"},
					"title": {"type": "string", "description": "New title (optional)"}
				}
			}`),
		},
		{
			Name:        "delete_link",
			Description: "Delete a link from an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "link_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-5 format"},
					"link_id": {"type": "string", "description": "Link UUID to delete"}
				}
			}`),
		},
		{
			Name:        "delete_issue",
			Description: "Delete a work-item (permanent)",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-5 format"}
				}
			}`),
		},
		{
			Name:        "add_relation",
			Description: "Add a relation to an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "target_issue_id", "relation_type"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-5 format"},
					"target_issue_id": {"type": "string", "description": "Target issue UUID or CATSTHOUGH-5 format"},
					"relation_type": {"type": "string", "description": "Relation type: blocker, relates_to, duplicate, child, parent"}
				}
			}`),
		},
		{
			Name:        "delete_relation",
			Description: "Remove a relation from an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "relation_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-5 format"},
					"relation_id": {"type": "string", "description": "Relation UUID to delete"}
				}
			}`),
		},
		{
			Name:        "create_attachment",
			Description: "Upload an attachment to an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "url"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-5 format"},
					"url": {"type": "string", "description": "URL of the attachment"},
					"title": {"type": "string", "description": "Attachment title (optional)"}
				}
			}`),
		},
		{
			Name:        "create_cycle",
			Description: "Create a new cycle",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id", "name"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID"},
					"name": {"type": "string", "description": "Cycle name"},
					"start_date": {"type": "string", "description": "Start date YYYY-MM-DD (optional)"},
					"end_date": {"type": "string", "description": "End date YYYY-MM-DD (optional)"}
				}
			}`),
		},
		{
			Name:        "create_module",
			Description: "Create a new module",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id", "name"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID"},
					"name": {"type": "string", "description": "Module name"},
					"description": {"type": "string", "description": "Module description (optional)"}
				}
			}`),
		},
		{
			Name:        "list_comments",
			Description: "List comments on an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID"}
				}
			}`),
		},
		{
			Name:        "list_labels",
			Description: "List all labels in a project",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID. Use list_projects first to get project IDs."}
				}
			}`),
		},
		{
			Name:        "reopen_issue",
			Description: "Reopen an archived/closed issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id"],
				"properties": {
					"workspace": {"type": "string"},
					"issue_id": {"type": "string"}
				}
			}`),
		},
		{
			Name:        "archive_issue",
			Description: "Archive an issue (soft delete)",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id"],
				"properties": {
					"workspace": {"type": "string"},
					"issue_id": {"type": "string"}
				}
			}`),
		},
		{
			Name:        "list_members",
			Description: "List project members for reassignment",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID. Use list_projects first to get project IDs."}
				}
			}`),
		},
		{
			Name:        "list_activities",
			Description: "List all activities (history) for an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-N format"}
				}
			}`),
		},
		{
			Name:        "create_label",
			Description: "Create a new label in a project",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id", "name", "color"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID"},
					"name": {"type": "string", "description": "Label name"},
					"color": {"type": "string", "description": "Color code (e.g., #FF0000)"}
				}
			}`),
		},
		{
			Name:        "update_comment",
			Description: "Update an existing comment",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "comment_id", "comment"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID"},
					"comment_id": {"type": "string", "description": "Comment UUID"},
					"comment": {"type": "string", "description": "Updated comment text"}
				}
			}`),
		},
		{
			Name:        "delete_comment",
			Description: "Delete a comment from an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id", "comment_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID"},
					"comment_id": {"type": "string", "description": "Comment UUID"}
				}
			}`),
		},
		{
			Name:        "list_cycles",
			Description: "List all cycles in a project",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID"}
				}
			}`),
		},
		{
			Name:        "get_cycle",
			Description: "Get a specific cycle by ID",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id", "cycle_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID"},
					"cycle_id": {"type": "string", "description": "Cycle UUID"}
				}
			}`),
		},
		{
			Name:        "list_modules",
			Description: "List all modules in a project",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID"}
				}
			}`),
		},
		{
			Name:        "get_module",
			Description: "Get a specific module by ID",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "project_id", "module_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"project_id": {"type": "string", "description": "Project UUID"},
					"module_id": {"type": "string", "description": "Module UUID"}
				}
			}`),
		},
		{
			Name:        "list_attachments",
			Description: "List attachments on an issue",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"required": ["workspace", "issue_id"],
				"properties": {
					"workspace": {"type": "string", "description": "Workspace slug (e.g., my-workspace)"},
					"issue_id": {"type": "string", "description": "Issue UUID or CATSTHOUGH-N format"}
				}
			}`),
		},
	}
}

func (s *Server) HandleRequest(ctx context.Context, req Request) Response {
	var result interface{}

	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]interface{}{
				"name":    "plane-mcp",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{},
		}
	case "tools/list":
		tools := make([]map[string]interface{}, len(s.tools))
		for i, t := range s.tools {
			tools[i] = map[string]interface{}{
				"name":         t.Name,
				"description":  t.Description,
				"inputSchema":   t.InputSchema,
			}
		}
		result = map[string]interface{}{
			"tools": tools,
		}
	case "get_defaults":
		result = s.getDefaults(ctx, req.Params, req.ID)
	case "list_projects":
		result = s.listProjects(ctx, req.Params, req.ID)
	case "get_project":
		result = s.getProject(ctx, req.Params, req.ID)
	case "list_issues":
		result = s.listIssues(ctx, req.Params, req.ID)
	case "get_issue":
		result = s.getIssue(ctx, req.Params, req.ID)
	case "create_issue":
		result = s.createIssue(ctx, req.Params, req.ID)
	case "update_issue":
		result = s.updateIssue(ctx, req.Params, req.ID)
	case "list_states":
		result = s.listStates(ctx, req.Params, req.ID)
	case "add_comment":
		result = s.addComment(ctx, req.Params, req.ID)
	case "list_comments":
		result = s.listComments(ctx, req.Params, req.ID)
	case "list_labels":
		result = s.listLabels(ctx, req.Params, req.ID)
	case "list_members":
		result = s.listMembers(ctx, req.Params, req.ID)
	case "archive_issue":
		result = s.archiveIssue(ctx, req.Params, req.ID)
	case "reopen_issue":
		result = s.reopenIssue(ctx, req.Params, req.ID)
	case "list_activities":
		result = s.listActivities(ctx, req.Params, req.ID)
	case "create_label":
		result = s.createLabel(ctx, req.Params, req.ID)
	case "update_comment":
		result = s.updateComment(ctx, req.Params, req.ID)
	case "delete_comment":
		result = s.deleteComment(ctx, req.Params, req.ID)
	case "list_cycles":
		result = s.listCycles(ctx, req.Params, req.ID)
	case "get_cycle":
		result = s.getCycle(ctx, req.Params, req.ID)
	case "list_modules":
		result = s.listModules(ctx, req.Params, req.ID)
	case "get_module":
		result = s.getModule(ctx, req.Params, req.ID)
	case "list_attachments":
		result = s.listAttachments(ctx, req.Params, req.ID)
	case "add_link":
		result = s.addLink(ctx, req.Params, req.ID)
	case "update_link":
		result = s.updateLink(ctx, req.Params, req.ID)
	case "delete_link":
		result = s.deleteLink(ctx, req.Params, req.ID)
	case "delete_issue":
		result = s.deleteIssue(ctx, req.Params, req.ID)
	case "add_relation":
		result = s.addRelation(ctx, req.Params, req.ID)
	case "delete_relation":
		result = s.deleteRelation(ctx, req.Params, req.ID)
	case "create_attachment":
		result = s.createAttachment(ctx, req.Params, req.ID)
	case "create_cycle":
		result = s.createCycle(ctx, req.Params, req.ID)
	case "create_module":
		result = s.createModule(ctx, req.Params, req.ID)
	case "tools/call":
		var callParams struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &Error{
					Code:    -32602,
					Message: fmt.Sprintf("Invalid params: %v", err),
				},
			}
		}

		args := callParams.Arguments
		if args == nil {
			args = json.RawMessage("{}")
		}

		switch callParams.Name {
		case "list_projects":
			result = s.listProjects(ctx, args, req.ID)
		case "get_project":
			result = s.getProject(ctx, args, req.ID)
		case "list_issues":
			result = s.listIssues(ctx, args, req.ID)
		case "get_issue":
			result = s.getIssue(ctx, args, req.ID)
		case "create_issue":
			result = s.createIssue(ctx, args, req.ID)
		case "update_issue":
			result = s.updateIssue(ctx, args, req.ID)
		case "list_states":
			result = s.listStates(ctx, args, req.ID)
		case "add_comment":
			result = s.addComment(ctx, args, req.ID)
		case "list_comments":
			result = s.listComments(ctx, args, req.ID)
		case "list_labels":
			result = s.listLabels(ctx, args, req.ID)
		case "list_members":
			result = s.listMembers(ctx, args, req.ID)
		case "archive_issue":
			result = s.archiveIssue(ctx, args, req.ID)
		case "reopen_issue":
			result = s.reopenIssue(ctx, args, req.ID)
		case "get_defaults":
			result = s.getDefaults(ctx, args, req.ID)
		case "add_link":
			result = s.addLink(ctx, args, req.ID)
		case "update_link":
			result = s.updateLink(ctx, args, req.ID)
		case "delete_link":
			result = s.deleteLink(ctx, args, req.ID)
		case "delete_issue":
			result = s.deleteIssue(ctx, args, req.ID)
		case "add_relation":
			result = s.addRelation(ctx, args, req.ID)
		case "delete_relation":
			result = s.deleteRelation(ctx, args, req.ID)
		case "create_attachment":
			result = s.createAttachment(ctx, args, req.ID)
		case "create_cycle":
			result = s.createCycle(ctx, args, req.ID)
		case "create_module":
			result = s.createModule(ctx, args, req.ID)
		case "list_activities":
			result = s.listActivities(ctx, args, req.ID)
		case "create_label":
			result = s.createLabel(ctx, args, req.ID)
		case "update_comment":
			result = s.updateComment(ctx, args, req.ID)
		case "delete_comment":
			result = s.deleteComment(ctx, args, req.ID)
		case "list_cycles":
			result = s.listCycles(ctx, args, req.ID)
		case "get_cycle":
			result = s.getCycle(ctx, args, req.ID)
		case "list_modules":
			result = s.listModules(ctx, args, req.ID)
		case "get_module":
			result = s.getModule(ctx, args, req.ID)
		case "list_attachments":
			result = s.listAttachments(ctx, args, req.ID)
		default:
			return Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &Error{
					Code:    -32601,
					Message: fmt.Sprintf("Unknown tool: %s", callParams.Name),
				},
			}
		}
	default:
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}

	if resp, ok := result.(Response); ok {
		return resp
	}

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) getProjectID(ctx context.Context, workspace, requestedProjectID string) (string, error) {
	if requestedProjectID != "" {
		resolved, err := s.resolveProjectByName(ctx, workspace, requestedProjectID)
		if err != nil {
			return "", err
		}
		return resolved, nil
	}

	if s.defaultProject != "" {
		resolved, err := s.resolveProjectByName(ctx, workspace, s.defaultProject)
		if err != nil {
			return "", err
		}
		return resolved, nil
	}

	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil || len(projects) == 0 {
		return "", fmt.Errorf("no projects found")
	}
	return projects[0].ID, nil
}

func (s *Server) resolveProjectByName(ctx context.Context, workspace, name string) (string, error) {
	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil {
		return "", fmt.Errorf("failed to fetch projects: %v", err)
	}

	name = strings.ToLower(name)

	for _, p := range projects {
		if strings.ToLower(p.Name) == name {
			return p.ID, nil
		}
	}

	for _, p := range projects {
		if strings.ToLower(p.ID) == name {
			return p.ID, nil
		}
	}

	return "", fmt.Errorf("project not found: %s", name)
}

func (s *Server) errorResponse(id interface{}, err error) Response {
	msg := "Internal error"
	if err != nil {
		msg = err.Error()
	}
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    -32603,
			Message: msg,
		},
	}
}

func (s *Server) getDefaults(ctx context.Context, args json.RawMessage, id interface{}) Response {
	debug("getDefaults: s.workspace=%s, s.defaultProject=%s", s.workspace, s.defaultProject)
	defaultProject := s.defaultProject
	if defaultProject == "" {
		projects, err := s.fetchProjects(ctx, s.workspace)
		if err == nil && len(projects) > 0 {
			defaultProject = projects[0].Name + " (" + projects[0].ID + ")"
		}
	} else {
		projectID, err := s.resolveProjectByName(ctx, s.workspace, s.defaultProject)
		if err == nil {
			defaultProject = s.defaultProject + " -> " + projectID
		}
	}
	return s.textResponse(fmt.Sprintf("workspace=%s, default_project=%s", s.workspace, defaultProject), id)
}

func (s *Server) resolveIssueIdentifier(ctx context.Context, workspace, identifier string) (string, error) {
	debug("resolveIssueIdentifier: workspace=%s, identifier=%s", workspace, identifier)
	if !strings.Contains(identifier, "-") {
		return identifier, nil
	}

	parts := strings.Split(identifier, "-")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid issue format: %s", identifier)
	}

	prefix := strings.Join(parts[:len(parts)-1], "-")
	seqNum := parts[len(parts)-1]

	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil {
		return "", fmt.Errorf("failed to fetch projects: %v", err)
	}

	for _, proj := range projects {
		if !strings.EqualFold(proj.Identifier, prefix) {
			continue
		}

		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/", s.baseURL, workspace, proj.ID)
		body, err := s.doRequest(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		if results, ok := body["results"].([]interface{}); ok {
			for _, r := range results {
				if rm, ok := r.(map[string]interface{}); ok {
					if sid, ok := rm["sequence_id"].(float64); ok {
						if fmt.Sprintf("%.0f", sid) == seqNum {
							if id, ok := rm["id"].(string); ok {
								return id, nil
							}
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("issue not found: %s", identifier)
}

func (s *Server) listProjects(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
	}
	if args != nil {
		json.Unmarshal(args, &params)
	}

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	debug("listProjects: params.Workspace=%s, s.workspace=%s, final=%s", params.Workspace, s.workspace, workspace)

	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil {
		if params.Workspace != "" && s.workspace != "" {
			debug("Fallback: trying s.workspace=%s", s.workspace)
			projects, err = s.fetchProjects(ctx, s.workspace)
		}
		if err != nil {
			return s.errorResponse(id, err)
		}
	}

	return s.textResponse(formatProjects(projects), id)
}

func (s *Server) getProject(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
	}
	json.Unmarshal(args, &params)

	project, err := s.fetchProject(ctx, params.Workspace, params.ProjectID)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatProject(project), id)
}

func (s *Server) listIssues(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id,omitempty"`
		Search   string `json:"search,omitempty"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	projectID, err := s.getProjectID(ctx, workspace, params.ProjectID)
	if err != nil {
		return s.errorResponse(id, err)
	}

	issues, err := s.fetchIssues(ctx, workspace, projectID, params.Search)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatIssues(issues), id)
}

func (s *Server) getIssue(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issueID := params.IssueID
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	issue, err := s.fetchIssue(ctx, workspace, issueID)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatIssue(issue), id)
}

func (s *Server) createIssue(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params CreateIssueParams
	json.Unmarshal(args, &params)

	if params.Workspace == "" {
		params.Workspace = s.workspace
	}

	issue, err := s.createNewIssue(ctx, params)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatIssue(issue), id)
}

func (s *Server) updateIssue(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params UpdateIssueParams
	json.Unmarshal(args, &params)

	if params.Workspace == "" {
		params.Workspace = s.workspace
	}

	if strings.Contains(params.IssueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, params.Workspace, params.IssueID)
		if err == nil {
			params.IssueID = resolved
		}
	}

	if params.Parent != "" && strings.Contains(params.Parent, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, params.Workspace, params.Parent)
		if err == nil {
			params.Parent = resolved
		}
	}

	issue, err := s.updateExistingIssue(ctx, params)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatIssue(issue), id)
}

func (s *Server) listStates(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
	}
	json.Unmarshal(args, &params)

	if params.ProjectID == "" {
		return s.errorResponse(id, fmt.Errorf("project_id is required. Use list_projects to get valid project IDs"))
	}

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	states, err := s.fetchStates(ctx, workspace, params.ProjectID)
	if err != nil {
		if params.ProjectID != "" {
			projects, projErr := s.fetchProjects(ctx, params.Workspace)
			if projErr == nil && len(projects) > 0 {
				var names []string
				for _, p := range projects {
					names = append(names, fmt.Sprintf("%s (%s)", p.Name, p.ID))
				}
				return s.errorResponse(id, fmt.Errorf("%v. Available projects: %s", err, strings.Join(names, ", ")))
			}
		}
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatStates(states), id)
}

func (s *Server) addComment(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
		Comment   string `json:"comment"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	comment, err := s.createComment(ctx, workspace, params.IssueID, params.Comment)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatComment(comment), id)
}

func (s *Server) addLink(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
		URL       string `json:"url"`
		Title     string `json:"title,omitempty"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issueID := params.IssueID
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	projectID, err := s.getProjectID(ctx, workspace, "")
	if err != nil {
		return s.errorResponse(id, err)
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/links/", s.baseURL, workspace, projectID, issueID)
	payload := map[string]interface{}{
		"url": params.URL,
	}
	if params.Title != "" {
		payload["title"] = params.Title
	}

	_, err = s.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Link added: %s (%s)", params.URL, params.Title), id)
}

func (s *Server) updateLink(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
		LinkID    string `json:"link_id"`
		URL       string `json:"url,omitempty"`
		Title     string `json:"title,omitempty"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issueID := params.IssueID
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	projectID, err := s.getProjectID(ctx, workspace, "")
	if err != nil {
		return s.errorResponse(id, err)
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/links/%s/", s.baseURL, workspace, projectID, issueID, params.LinkID)
	payload := map[string]interface{}{}
	if params.URL != "" {
		payload["url"] = params.URL
	}
	if params.Title != "" {
		payload["title"] = params.Title
	}

	_, err = s.doRequest(ctx, "PATCH", url, payload)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Link updated: %s (%s)", params.URL, params.Title), id)
}

func (s *Server) deleteLink(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
		LinkID    string `json:"link_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issueID := params.IssueID
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	projectID, err := s.getProjectID(ctx, workspace, "")
	if err != nil {
		return s.errorResponse(id, err)
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/links/%s/", s.baseURL, workspace, projectID, issueID, params.LinkID)

	_, err = s.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Link deleted: %s", params.LinkID), id)
}

func (s *Server) deleteIssue(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issueID := params.IssueID
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	projectID, err := s.getProjectID(ctx, workspace, "")
	if err != nil {
		return s.errorResponse(id, err)
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/", s.baseURL, workspace, projectID, issueID)

	_, err = s.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Issue deleted: %s", issueID), id)
}

func (s *Server) addRelation(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace      string `json:"workspace"`
		IssueID        string `json:"issue_id"`
		TargetIssueID  string `json:"target_issue_id"`
		RelationType   string `json:"relation_type"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issueID := params.IssueID
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	targetID := params.TargetIssueID
	if strings.Contains(targetID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, targetID)
		if err == nil {
			targetID = resolved
		}
	}

	_, err := s.createRelation(ctx, workspace, issueID, targetID, params.RelationType)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Relation created: %s -> %s (%s)", issueID, targetID, params.RelationType), id)
}

func (s *Server) deleteRelation(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace  string `json:"workspace"`
		IssueID    string `json:"issue_id"`
		RelationID string `json:"relation_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issueID := params.IssueID
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	projectID, err := s.getProjectID(ctx, workspace, "")
	if err != nil {
		return s.errorResponse(id, err)
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/relations/%s/", s.baseURL, workspace, projectID, issueID, params.RelationID)

	_, err = s.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Relation deleted: %s", params.RelationID), id)
}

func (s *Server) createAttachment(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
		URL       string `json:"url"`
		Title     string `json:"title,omitempty"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issueID := params.IssueID
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	projectID, err := s.getProjectID(ctx, workspace, "")
	if err != nil {
		return s.errorResponse(id, err)
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/attachments/", s.baseURL, workspace, projectID, issueID)
	payload := map[string]interface{}{
		"url": params.URL,
	}
	if params.Title != "" {
		payload["title"] = params.Title
	}

	_, err = s.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Attachment added: %s (%s)", params.URL, params.Title), id)
}

func (s *Server) createCycle(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace  string `json:"workspace"`
		ProjectID  string `json:"project_id"`
		Name       string `json:"name"`
		StartDate  string `json:"start_date,omitempty"`
		EndDate    string `json:"end_date,omitempty"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	projectID, err := s.getProjectID(ctx, workspace, params.ProjectID)
	if err != nil {
		return s.errorResponse(id, err)
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/cycles/", s.baseURL, workspace, projectID)
	payload := map[string]interface{}{
		"name": params.Name,
	}
	if params.StartDate != "" {
		payload["start_date"] = params.StartDate
	}
	if params.EndDate != "" {
		payload["end_date"] = params.EndDate
	}

	_, err = s.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Cycle created: %s", params.Name), id)
}

func (s *Server) createModule(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace   string `json:"workspace"`
		ProjectID   string `json:"project_id"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	projectID, err := s.getProjectID(ctx, workspace, params.ProjectID)
	if err != nil {
		return s.errorResponse(id, err)
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/modules/", s.baseURL, workspace, projectID)
	payload := map[string]interface{}{
		"name": params.Name,
	}
	if params.Description != "" {
		payload["description"] = params.Description
	}

	_, err = s.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Module created: %s", params.Name), id)
}

func (s *Server) listComments(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	comments, err := s.fetchComments(ctx, workspace, params.IssueID)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatComments(comments), id)
}

func (s *Server) listLabels(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	labels, err := s.fetchLabels(ctx, workspace, params.ProjectID)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatLabels(labels), id)
}

func (s *Server) reopenIssue(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issue, err := s.updateIssueField(ctx, workspace, params.IssueID, map[string]interface{}{
		"archived_at": nil,
	})
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatIssue(issue), id)
}

func (s *Server) archiveIssue(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	issue, err := s.updateIssueField(ctx, workspace, params.IssueID, map[string]interface{}{
		"archived_at": "now",
	})
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(fmt.Sprintf("Issue archived: %s", issue.Name), id)
}

func (s *Server) listMembers(ctx context.Context, args json.RawMessage, id interface{}) Response {
	var params struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
	}
	json.Unmarshal(args, &params)

	workspace := params.Workspace
	if workspace == "" {
		workspace = s.workspace
	}

	members, err := s.fetchMembers(ctx, workspace, params.ProjectID)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.textResponse(formatMembers(members), id)
}

func (s *Server) textResponse(text string, id interface{}) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": text,
				},
			},
		},
	}
}

type Project struct {
	ID          string `mapstructure:"id"`
	Name        string `mapstructure:"name"`
	Identifier  string `mapstructure:"identifier"`
	Description string `mapstructure:"description"`
	CreatedAt   string `mapstructure:"created_at"`
}

type State struct {
	ID    string `mapstructure:"id"`
	Name  string `mapstructure:"name"`
	Group string `mapstructure:"group"`
}

type Issue struct {
	ID          string   `mapstructure:"id"`
	Name        string   `mapstructure:"name"`
	Description string   `mapstructure:"description_html"`
	State       string   `mapstructure:"state"`
	Priority    string   `mapstructure:"priority"`
	ProjectID   string   `mapstructure:"project"`
	Parent      string   `mapstructure:"parent"`
	Assignees    []string `mapstructure:"assignees"`
	AssigneeNames []string `mapstructure:"-"`
	Labels      []string `mapstructure:"labels"`
	LabelNames  []string `mapstructure:"-"`
	StartDate   string   `mapstructure:"start_date"`
	TargetDate  string   `mapstructure:"target_date"`
	CreatedAt   string   `mapstructure:"created_at"`
	Links       []Link   `mapstructure:"-"`
	Relations   []Relation `mapstructure:"-"`
}

type Link struct {
	ID    string `mapstructure:"id"`
	Title string `mapstructure:"title"`
	URL   string `mapstructure:"url"`
}

type Relation struct {
	ID     string `mapstructure:"id"`
	Type   string `mapstructure:"relation_type"`
	Target string `mapstructure:"target_issue"`
}

type CreateIssueParams struct {
	Workspace string   `json:"workspace"`
	ProjectID string   `json:"project_id"`
	Name      string   `json:"name"`
	Desc      string   `json:"description,omitempty"`
	State     string   `json:"state,omitempty"`
	Priority  string   `json:"priority,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Parent    string   `json:"parent,omitempty"`
}

type UpdateIssueParams struct {
	Workspace   string   `json:"workspace"`
	IssueID     string   `json:"issue_id"`
	Name        string   `json:"name,omitempty"`
	Desc        string   `json:"description,omitempty"`
	State       string   `json:"state,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Assignees   []string `json:"assignees,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	StartDate   string   `json:"start_date,omitempty"`
	TargetDate  string   `json:"target_date,omitempty"`
}

type Comment struct {
	ID          string `mapstructure:"id"`
	CommentHTML string `mapstructure:"comment_html"`
	CreatedAt   string `mapstructure:"created_at"`
}

type Label struct {
	ID    string `mapstructure:"id"`
	Name  string `mapstructure:"name"`
	Color string `mapstructure:"color"`
}

type Member struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type Activity struct {
	ID        string `mapstructure:"id"`
	Verb      string `mapstructure:"verb"`
	Field     string `mapstructure:"field"`
	OldValue  string `mapstructure:"old_value"`
	NewValue  string `mapstructure:"new_value"`
	Comment   string `mapstructure:"comment"`
	CreatedAt string `mapstructure:"created_at"`
	Actor     string `mapstructure:"actor"`
}

type Cycle struct {
	ID        string `mapstructure:"id"`
	Name      string `mapstructure:"name"`
	Status    string `mapstructure:"status"`
	StartDate string `mapstructure:"start_date"`
	EndDate   string `mapstructure:"end_date"`
}

type Module struct {
	ID        string `mapstructure:"id"`
	Name      string `mapstructure:"name"`
	Status    string `mapstructure:"status"`
}

func (s *Server) fetchProjects(ctx context.Context, workspace string) ([]Project, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/", s.baseURL, workspace)
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if apiErr, ok := body["error"].(string); ok {
		return nil, fmt.Errorf("API error: %s", apiErr)
	}

	resultsData, ok := body["results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("workspace '%s' not found or no access", workspace)
	}

	var results []Project
	for _, r := range resultsData {
		if m, ok := r.(map[string]interface{}); ok {
			p := Project{
				ID:          getString(m, "id"),
				Name:        getString(m, "name"),
				Identifier:  getString(m, "identifier"),
				Description: getString(m, "description_text"),
				CreatedAt:   getString(m, "created_at"),
			}
			results = append(results, p)
		}
	}
	return results, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func (s *Server) fetchProject(ctx context.Context, workspace, projectID string) (Project, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/", s.baseURL, workspace, projectID)
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return Project{}, err
	}

	var project Project
	if err := mapstructure.Decode(body, &project); err != nil {
		return Project{}, err
	}
	return project, nil
}

func (s *Server) fetchIssues(ctx context.Context, workspace, projectID, search string) ([]Issue, error) {
	var url string
	if projectID != "" {
		url = fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/", s.baseURL, workspace, projectID)
	} else {
		projects, _ := s.fetchProjects(ctx, workspace)
		if len(projects) == 0 {
			return []Issue{}, nil
		}
		url = fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/", s.baseURL, workspace, projects[0].ID)
	}
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resultsData, ok := body["results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected results format")
	}

	var results []Issue
	for _, r := range resultsData {
		if m, ok := r.(map[string]interface{}); ok {
			i := Issue{
				ID:          getString(m, "id"),
				Name:        getString(m, "name"),
				Description: getString(m, "description_html"),
				State:       getString(m, "state"),
				Priority:    getString(m, "priority"),
				ProjectID:   getString(m, "project"),
				CreatedAt:   getString(m, "created_at"),
			}
			if labels, ok := m["labels"].([]interface{}); ok {
				for _, l := range labels {
					if ls, ok := l.(string); ok {
						i.Labels = append(i.Labels, ls)
					}
				}
			}
			results = append(results, i)
		}
	}

	if search != "" {
		searchLower := strings.ToLower(search)
		var filtered []Issue
		for _, issue := range results {
			if strings.Contains(strings.ToLower(issue.Name), searchLower) ||
				strings.Contains(strings.ToLower(issue.Description), searchLower) {
				filtered = append(filtered, issue)
			}
		}
		results = filtered
	}

	return results, nil
}

func (s *Server) fetchIssue(ctx context.Context, workspace, issueID string) (Issue, error) {
	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil {
		return Issue{}, err
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/", s.baseURL, workspace, p.ID)
		body, err := s.doRequest(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		var result struct {
			Results []Issue
		}
		mapstructure.Decode(body, &result)

		for _, issue := range result.Results {
			if issue.ID == issueID {
				linksURL := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/links/", s.baseURL, workspace, p.ID, issueID)
				linksBody, err := s.doRequest(ctx, "GET", linksURL, nil)
				if err == nil {
					if linksResults, ok := linksBody["results"].([]interface{}); ok {
						for _, lr := range linksResults {
							if lm, ok := lr.(map[string]interface{}); ok {
								issue.Links = append(issue.Links, Link{
									ID:    getString(lm, "id"),
									Title: getString(lm, "title"),
									URL:   getString(lm, "url"),
								})
							}
						}
					}
				}

				relationsURL := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/relations/", s.baseURL, workspace, p.ID, issueID)
				relationsBody, err := s.doRequest(ctx, "GET", relationsURL, nil)
				if err == nil {
					if relationsResults, ok := relationsBody["results"].([]interface{}); ok {
						for _, rr := range relationsResults {
							if rm, ok := rr.(map[string]interface{}); ok {
								issue.Relations = append(issue.Relations, Relation{
									ID:     getString(rm, "id"),
									Type:   getString(rm, "relation_type"),
									Target: getString(rm, "target_issue"),
								})
							}
						}
					}
				}

				labelsSlice, _ := s.fetchLabels(ctx, workspace, p.ID)
				labelMap := make(map[string]string)
				for _, l := range labelsSlice {
					labelMap[l.ID] = l.Name
				}
				for _, labelID := range issue.Labels {
					if name, ok := labelMap[labelID]; ok {
						issue.LabelNames = append(issue.LabelNames, name)
					} else {
						issue.LabelNames = append(issue.LabelNames, labelID)
					}
				}

				membersSlice, _ := s.fetchMembers(ctx, workspace, p.ID)
				memberMap := make(map[string]string)
				for _, m := range membersSlice {
					memberName := m.FirstName + " " + m.LastName
					if memberName == " " {
						memberName = m.Email
					}
					memberMap[m.ID] = strings.TrimSpace(memberName)
				}
				for _, assigneeID := range issue.Assignees {
					if name, ok := memberMap[assigneeID]; ok {
						issue.AssigneeNames = append(issue.AssigneeNames, name)
					} else {
						issue.AssigneeNames = append(issue.AssigneeNames, assigneeID)
					}
				}

				return issue, nil
			}
		}
	}
	return Issue{}, fmt.Errorf("issue not found")
}

func (s *Server) fetchStates(ctx context.Context, workspace, projectID string) ([]State, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/states/", s.baseURL, workspace, projectID)
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resultsData, ok := body["results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected results format")
	}

	var results []State
	for _, r := range resultsData {
		if m, ok := r.(map[string]interface{}); ok {
			st := State{
				ID:    getString(m, "id"),
				Name:  getString(m, "name"),
				Group: getString(m, "group"),
			}
			results = append(results, st)
		}
	}
	return results, nil
}

func (s *Server) fetchComments(ctx context.Context, workspace, issueID string) ([]Comment, error) {
	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/comments/", s.baseURL, workspace, p.ID, issueID)
		body, err := s.doRequest(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resultsData, ok := body["results"].([]interface{})
		if !ok {
			continue
		}

		var results []Comment
		for _, r := range resultsData {
			if m, ok := r.(map[string]interface{}); ok {
				c := Comment{
					ID:          getString(m, "id"),
					CommentHTML: getString(m, "comment_html"),
					CreatedAt:   getString(m, "created_at"),
				}
				results = append(results, c)
			}
		}
		if len(results) > 0 {
			return results, nil
		}
	}
	return []Comment{}, nil
}

func (s *Server) createComment(ctx context.Context, workspace, issueID, comment string) (Comment, error) {
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil {
		return Comment{}, err
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/comments/", s.baseURL, workspace, p.ID, issueID)
		payload := map[string]interface{}{
			"comment_html": "<p>" + comment + "</p>",
		}

		body, err := s.doRequest(ctx, "POST", url, payload)
		if err != nil {
			continue
		}

		var result Comment
		if err := mapstructure.Decode(body, &result); err != nil {
			continue
		}
		if result.ID != "" {
			return result, nil
		}
	}
	return Comment{}, fmt.Errorf("failed to create comment")
}

func (s *Server) fetchLabels(ctx context.Context, workspace, projectID string) ([]Label, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/labels/", s.baseURL, workspace, projectID)
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resultsData, ok := body["results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected results format")
	}

	var results []Label
	for _, r := range resultsData {
		if m, ok := r.(map[string]interface{}); ok {
			l := Label{
				ID:    getString(m, "id"),
				Name:  getString(m, "name"),
				Color: getString(m, "color"),
			}
			results = append(results, l)
		}
	}
	return results, nil
}

func (s *Server) resolveLabelNamesToUUIDs(ctx context.Context, workspace, projectID string, names []string) ([]string, error) {
	labels, err := s.fetchLabels(ctx, workspace, projectID)
	if err != nil {
		return nil, err
	}

	var uuids []string
	for _, name := range names {
		found := false
		for _, label := range labels {
			if label.Name == name {
				uuids = append(uuids, label.ID)
				found = true
				break
			}
		}
		if !found {
			uuids = append(uuids, name)
		}
	}
	return uuids, nil
}

func (s *Server) fetchMembers(ctx context.Context, workspace, projectID string) ([]Member, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/members/", s.baseURL, workspace, projectID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", s.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// API returns array directly
	var members []Member
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, err
	}
	return members, nil
}

func (s *Server) updateIssueField(ctx context.Context, workspace, issueID string, payload map[string]interface{}) (Issue, error) {
	if strings.Contains(issueID, "-") {
		resolved, err := s.resolveIssueIdentifier(ctx, workspace, issueID)
		if err == nil {
			issueID = resolved
		}
	}

	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil {
		return Issue{}, err
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/", s.baseURL, workspace, p.ID, issueID)
		body, err := s.doRequest(ctx, "PATCH", url, payload)
		if err != nil {
			continue
		}

		var issue Issue
		if err := mapstructure.Decode(body, &issue); err != nil {
			continue
		}
		if issue.ID != "" {
			return issue, nil
		}
	}
	return Issue{}, fmt.Errorf("issue not found")
}

func (s *Server) createRelation(ctx context.Context, workspace, issueID, targetID, relationType string) (Relation, error) {
	projects, err := s.fetchProjects(ctx, workspace)
	if err != nil {
		return Relation{}, err
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/relations/", s.baseURL, workspace, p.ID, issueID)
		payload := map[string]interface{}{
			"target_issue_id": targetID,
			"relation_type":  relationType,
		}

		body, err := s.doRequest(ctx, "POST", url, payload)
		if err != nil {
			continue
		}

		var rel Relation
		if err := mapstructure.Decode(body, &rel); err != nil {
			continue
		}
		if rel.ID != "" {
			return rel, nil
		}
	}
	return Relation{}, fmt.Errorf("failed to create relation")
}

func (s *Server) createNewIssue(ctx context.Context, params CreateIssueParams) (Issue, error) {
	projectID, err := s.getProjectID(ctx, params.Workspace, params.ProjectID)
	if err != nil {
		return Issue{}, err
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/", s.baseURL, params.Workspace, projectID)

	payload := map[string]interface{}{
		"name": params.Name,
	}
	if params.Desc != "" {
		payload["description_html"] = params.Desc
	}
	if params.State != "" {
		payload["state"] = params.State
	}
	if params.Priority != "" {
		payload["priority"] = params.Priority
	}
	if params.Parent != "" {
		parentID := params.Parent
		if strings.Contains(parentID, "-") {
			resolved, err := s.resolveIssueIdentifier(ctx, params.Workspace, parentID)
			if err == nil {
				parentID = resolved
			}
		}
		payload["parent"] = parentID
	}
	if len(params.Assignees) > 0 {
		payload["assignees"] = params.Assignees
	}
	if len(params.Labels) > 0 {
		payload["labels"] = params.Labels
	}

	body, err := s.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return Issue{}, err
	}

	var issue Issue
	if err := mapstructure.Decode(body, &issue); err != nil {
		return Issue{}, err
	}
	return issue, nil
}

func (s *Server) updateExistingIssue(ctx context.Context, params UpdateIssueParams) (Issue, error) {
	projects, err := s.fetchProjects(ctx, params.Workspace)
	if err != nil {
		return Issue{}, err
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/", s.baseURL, params.Workspace, p.ID, params.IssueID)
		payload := map[string]interface{}{}
		if params.Name != "" {
			payload["name"] = params.Name
		}
		if params.Desc != "" {
			payload["description_html"] = params.Desc
		}
		if params.State != "" {
			payload["state"] = params.State
		}
		if params.Priority != "" {
			payload["priority"] = params.Priority
		}
		if len(params.Assignees) > 0 {
			payload["assignees"] = params.Assignees
		}
		if len(params.Labels) > 0 {
			labelUUIDs, err := s.resolveLabelNamesToUUIDs(ctx, params.Workspace, p.ID, params.Labels)
			if err != nil {
				continue
			}
			payload["labels"] = labelUUIDs
		}
		if params.Parent != "" {
			payload["parent"] = params.Parent
		}
		if params.StartDate != "" {
			payload["start_date"] = params.StartDate
		}
		if params.TargetDate != "" {
			payload["target_date"] = params.TargetDate
		}

		body, err := s.doRequest(ctx, "PATCH", url, payload)
		if err != nil {
			continue
		}

		var issue Issue
		if err := mapstructure.Decode(body, &issue); err != nil {
			continue
		}
		if issue.ID != "" {
			return issue, nil
		}
	}
	return Issue{}, fmt.Errorf("issue not found")
}

func (s *Server) doRequest(ctx context.Context, method, url string, payload interface{}) (map[string]interface{}, error) {
	debug("HTTP %s %s", method, url)
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
		debug("Payload: %s", string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	debug("Response status: %d", resp.StatusCode)

	if resp.StatusCode == 204 {
		return map[string]interface{}{}, nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode error: %v, status: %d, url: %s", err, resp.StatusCode, url)
	}
	return result, nil
}

func formatProjects(projects []Project) string {
	var b strings.Builder
	b.WriteString("# Projects\n\n")
	b.WriteString(fmt.Sprintf("Default workspace: `%s`\n\n", getEnv("PLANE_WORKSPACE", "")))
	for _, p := range projects {
		b.WriteString(fmt.Sprintf("- **%s** (%s) - ID: `%s`\n", p.Name, p.Identifier, p.ID))
	}
	b.WriteString("\nUse project ID (UUID) for operations, not identifier.")
	return b.String()
}

func formatProject(p Project) string {
	return fmt.Sprintf("# Project: %s\n\n**Identifier:** %s\n\n**Description:** %s\n\n**Created:** %s\n", p.Name, p.Identifier, p.Description, p.CreatedAt)
}

func formatIssues(issues []Issue) string {
	var b strings.Builder
	b.WriteString("# Issues\n\n")
	for _, i := range issues {
		b.WriteString(fmt.Sprintf("- **[%s](%s)** [%s] %s\n", i.Name, i.ID, i.State, i.Priority))
	}
	return b.String()
}

func formatIssue(i Issue) string {
	assignees := ""
	if len(i.AssigneeNames) > 0 {
		assignees = fmt.Sprintf("\n**Assignees:** %v", i.AssigneeNames)
	}
	labels := ""
	if len(i.LabelNames) > 0 {
		labels = fmt.Sprintf("\n**Labels:** %v", i.LabelNames)
	}
	dates := ""
	if i.StartDate != "" || i.TargetDate != "" {
		dates = fmt.Sprintf("\n**Dates:** %s → %s", i.StartDate, i.TargetDate)
	}
	parent := ""
	if i.Parent != "" {
		parent = fmt.Sprintf("\n**Parent:** %s", i.Parent)
	}
	links := ""
	if len(i.Links) > 0 {
		links = "\n**Links:**\n"
		for _, l := range i.Links {
			links += fmt.Sprintf("- [%s](%s)\n", l.Title, l.URL)
		}
	}
	relations := ""
	if len(i.Relations) > 0 {
		relations = "\n**Relations:**\n"
		for _, r := range i.Relations {
			relations += fmt.Sprintf("- %s → %s\n", r.Type, r.Target)
		}
	}
	return fmt.Sprintf("# Issue: %s\n\n**ID:** %s\n**State:** %s\n**Priority:** %s%s%s%s%s%s%s\n\n## Description\n%s\n\n**Created:** %s\n", i.Name, i.ID, i.State, i.Priority, assignees, labels, dates, parent, links, relations, i.Description, i.CreatedAt)
}

func formatLabels(labels []Label) string {
	if len(labels) == 0 {
		return "# Labels\n\nNo labels in project."
	}
	var b strings.Builder
	b.WriteString("# Labels\n\n")
	for _, l := range labels {
		b.WriteString(fmt.Sprintf("- **%s** #%s\n", l.Name, l.Color))
	}
	return b.String()
}

func formatMembers(members []Member) string {
	if len(members) == 0 {
		return "# Members\n\nNo members in project."
	}
	var b strings.Builder
	b.WriteString("# Members\n\n")
	for _, m := range members {
		var name string
		if m.FirstName != "" || m.LastName != "" {
			name = strings.TrimSpace(m.FirstName + " " + m.LastName)
		} else {
			name = m.Email
		}
		b.WriteString(fmt.Sprintf("- **%s** (%s) - UUID: %s\n", name, m.Email, m.ID))
	}
	return b.String()
}

func formatStates(states []State) string {
	var b strings.Builder
	b.WriteString("# States\n\n")
	for _, st := range states {
		group := ""
		switch st.Group {
		case "backlog":
			group = "📥 Backlog"
		case "unstarted":
			group = "📋 Todo"
		case "started":
			group = "🔄 In Progress"
		case "completed":
			group = "✅ Done"
		case "cancelled":
			group = "❌ Cancelled"
		}
		b.WriteString(fmt.Sprintf("- **%s** %s (id: %s)\n", st.Name, group, st.ID))
	}
	return b.String()
}

func formatComment(c Comment) string {
	commentText := strings.TrimSuffix(strings.TrimPrefix(c.CommentHTML, "<p>"), "</p>")
	return fmt.Sprintf("# Comment\n\n%s\n\n**ID:** %s\n**Created:** %s\n", commentText, c.ID, c.CreatedAt)
}

func formatComments(comments []Comment) string {
	if len(comments) == 0 {
		return "# Comments\n\nNo comments yet."
	}
	var b strings.Builder
	b.WriteString("# Comments\n\n")
	for _, c := range comments {
		commentText := strings.TrimSuffix(strings.TrimPrefix(c.CommentHTML, "<p>"), "</p>")
		b.WriteString(fmt.Sprintf("---\n%s\n**ID:** %s | **Created:** %s\n", commentText, c.ID, c.CreatedAt))
	}
	return b.String()
}

const DEBUG = false

var logFile *os.File

func debug(format string, args ...interface{}) {
	if DEBUG && logFile != nil {
		logFile.WriteString(fmt.Sprintf("[DEBUG] "+format+"\n", args...))
		logFile.Sync()
	}
}

func openLog() {
	var err error
	logFile, err = os.OpenFile("/tmp/plane-mcp-debug.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logFile = os.Stderr
	}
}

func closeLog() {
	if logFile != nil && logFile != os.Stderr {
		logFile.Close()
	}
}

func main() {
	openLog()
	defer closeLog()

	if os.Getenv("OPENCODE") == "" || getEnv("PLANE_WORKSPACE", "") == "" {
		loadEnv()
	}

	debug("Starting plane-mcp")
	debug("PLANE_WORKSPACE=%s", getEnv("PLANE_WORKSPACE", ""))
	debug("PLANE_API_KEY=%s", getEnv("PLANE_API_KEY", ""))
	debug("PLANE_BASE_URL=%s", getEnv("PLANE_BASE_URL", ""))

	workspace := getEnv("PLANE_WORKSPACE", "")
	defaultProject := getEnv("PLANE_DEFAULT_PROJECT", "")
	apiKey := getEnv("PLANE_API_KEY", "")
	baseURL := getEnv("PLANE_BASE_URL", "http://localhost:8282")

	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "Required: PLANE_API_KEY\n")
		fmt.Fprintf(os.Stderr, "Set it in .env file or PLANE_API_KEY environment variable\n")
		os.Exit(1)
	}

	server := NewServer(workspace, defaultProject, apiKey, baseURL)

	if isTerminal() {
		runStdio(server)
	} else {
		runPipe(server)
	}
}

func loadEnv() {
	paths := []string{
		".env",
		filepath.Join(getCurrentDir(), ".env"),
		filepath.Join(userHomeDir(), ".plane-mcp", ".env"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			godotenv.Overload(path)
			return
		}
	}
}

func getCurrentDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func userHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		return "."
	}
	return usr.HomeDir
}

func isTerminal() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func runPipe(server *Server) {
	decoder := json.NewDecoder(os.Stdin)
	for decoder.More() {
		var req Request
		if err := decoder.Decode(&req); err != nil {
			debug("Failed to decode request: %v", err)
			continue
		}
		debug("Incoming request: method=%s", req.Method)
		resp := server.HandleRequest(context.Background(), req)
		json.NewEncoder(os.Stdout).Encode(resp)
	}
	debug("runPipe: no more input")
}

func runStdio(server *Server) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		debug("Incoming request: %s", scanner.Bytes())
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			debug("Failed to parse request: %v", err)
			continue
		}
		debug("Handling method: %s", req.Method)
		resp := server.HandleRequest(context.Background(), req)
		json.NewEncoder(os.Stdout).Encode(resp)
		debug("Sent response")
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func (s *Server) listActivities(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	issueID, err := s.resolveIssueIdentifier(ctx, args.Workspace, args.IssueID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Issue not found: %s", args.IssueID))
	}

	projects, err := s.fetchProjects(ctx, args.Workspace)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to fetch projects: %v", err))
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/activities/", s.baseURL, args.Workspace, p.ID, issueID)
		body, err := s.doRequest(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resultsData, ok := body["results"].([]interface{})
		if !ok {
			continue
		}

		var activities []Activity
		for _, r := range resultsData {
			if m, ok := r.(map[string]interface{}); ok {
				a := Activity{
					ID:        getString(m, "id"),
					Verb:      getString(m, "verb"),
					Field:     getString(m, "field"),
					OldValue:  getString(m, "old_value"),
					NewValue:  getString(m, "new_value"),
					Comment:   getString(m, "comment"),
					CreatedAt: getString(m, "created_at"),
					Actor:     getString(m, "actor"),
				}
				activities = append(activities, a)
			}
		}
		if len(activities) > 0 {
			return successResponse(id, activities)
		}
	}
	return successResponse(id, []Activity{})
}

func (s *Server) createLabel(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
		Color     string `json:"color"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	projectID, err := s.getProjectID(ctx, args.Workspace, args.ProjectID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Project not found: %s", args.ProjectID))
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/labels/", s.baseURL, args.Workspace, projectID)
	payload := map[string]interface{}{
		"name":  args.Name,
		"color": args.Color,
	}

	body, err := s.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to create label: %v", err))
	}

	return successResponse(id, body)
}

func (s *Server) updateComment(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace  string `json:"workspace"`
		IssueID    string `json:"issue_id"`
		CommentID  string `json:"comment_id"`
		Comment    string `json:"comment"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	issueID, err := s.resolveIssueIdentifier(ctx, args.Workspace, args.IssueID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Issue not found: %s", args.IssueID))
	}

	projects, err := s.fetchProjects(ctx, args.Workspace)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to fetch projects: %v", err))
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/comments/%s/", s.baseURL, args.Workspace, p.ID, issueID, args.CommentID)
		payload := map[string]interface{}{
			"comment_html": "<p>" + args.Comment + "</p>",
		}

		body, err := s.doRequest(ctx, "PATCH", url, payload)
		if err != nil {
			continue
		}

		if getString(body, "id") != "" {
			return successResponse(id, body)
		}
	}
	return errorResponse(id, "Failed to update comment")
}

func (s *Server) deleteComment(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
		CommentID string `json:"comment_id"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	issueID, err := s.resolveIssueIdentifier(ctx, args.Workspace, args.IssueID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Issue not found: %s", args.IssueID))
	}

	projects, err := s.fetchProjects(ctx, args.Workspace)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to fetch projects: %v", err))
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/comments/%s/", s.baseURL, args.Workspace, p.ID, issueID, args.CommentID)

		_, err := s.doRequest(ctx, "DELETE", url, nil)
		if err == nil {
			return successResponse(id, map[string]string{"status": "deleted"})
		}
	}
	return errorResponse(id, "Failed to delete comment")
}

func (s *Server) listCycles(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	projectID, err := s.getProjectID(ctx, args.Workspace, args.ProjectID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Project not found: %s", args.ProjectID))
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/cycles/", s.baseURL, args.Workspace, projectID)
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to list cycles: %v", err))
	}

	resultsData, _ := body["results"].([]interface{})
	var cycles []Cycle
	for _, r := range resultsData {
		if m, ok := r.(map[string]interface{}); ok {
			c := Cycle{
				ID:        getString(m, "id"),
				Name:      getString(m, "name"),
				Status:    getString(m, "status"),
				StartDate: getString(m, "start_date"),
				EndDate:   getString(m, "end_date"),
			}
			cycles = append(cycles, c)
		}
	}

	return successResponse(id, cycles)
}

func (s *Server) getCycle(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
		CycleID   string `json:"cycle_id"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	projectID, err := s.getProjectID(ctx, args.Workspace, args.ProjectID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Project not found: %s", args.ProjectID))
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/cycles/%s/", s.baseURL, args.Workspace, projectID, args.CycleID)
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to get cycle: %v", err))
	}

	return successResponse(id, body)
}

func (s *Server) listModules(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	projectID, err := s.getProjectID(ctx, args.Workspace, args.ProjectID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Project not found: %s", args.ProjectID))
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/modules/", s.baseURL, args.Workspace, projectID)
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to list modules: %v", err))
	}

	resultsData, _ := body["results"].([]interface{})
	var modules []Module
	for _, r := range resultsData {
		if m, ok := r.(map[string]interface{}); ok {
			mod := Module{
				ID:     getString(m, "id"),
				Name:   getString(m, "name"),
				Status: getString(m, "status"),
			}
			modules = append(modules, mod)
		}
	}

	return successResponse(id, modules)
}

func (s *Server) getModule(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace string `json:"workspace"`
		ProjectID string `json:"project_id"`
		ModuleID  string `json:"module_id"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	projectID, err := s.getProjectID(ctx, args.Workspace, args.ProjectID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Project not found: %s", args.ProjectID))
	}

	url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/modules/%s/", s.baseURL, args.Workspace, projectID, args.ModuleID)
	body, err := s.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to get module: %v", err))
	}

	return successResponse(id, body)
}

func (s *Server) listAttachments(ctx context.Context, params json.RawMessage, id interface{}) map[string]interface{} {
	var args struct {
		Workspace string `json:"workspace"`
		IssueID   string `json:"issue_id"`
	}
	if err := json.Unmarshal(params, &args); err != nil {
		return errorResponse(id, "Invalid params")
	}

	if args.Workspace == "" {
		args.Workspace = s.workspace
	}

	issueID, err := s.resolveIssueIdentifier(ctx, args.Workspace, args.IssueID)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Issue not found: %s", args.IssueID))
	}

	projects, err := s.fetchProjects(ctx, args.Workspace)
	if err != nil {
		return errorResponse(id, fmt.Sprintf("Failed to fetch projects: %v", err))
	}

	for _, p := range projects {
		url := fmt.Sprintf("%s/api/v1/workspaces/%s/projects/%s/work-items/%s/attachments/", s.baseURL, args.Workspace, p.ID, issueID)
		body, err := s.doRequest(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resultsData, ok := body["results"].([]interface{})
		if !ok {
			continue
		}

		return successResponse(id, resultsData)
	}
	return successResponse(id, []interface{}{})
}

func errorResponse(id interface{}, msg string) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    -32600,
			"message": msg,
		},
	}
}

func successResponse(id interface{}, result interface{}) map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}
