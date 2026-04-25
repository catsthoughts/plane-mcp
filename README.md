# Plane MCP Server

[MCP](https://modelcontextprotocol.info/) server for integrating with [Plane](https://plane.so) — open-source project management system.

## Features

- **Projects** — list and view project details
- **Issues** — create, view, update tickets (priority, status, assignees, labels, dates, parent, links, relations)
- **States** — view workflow states
- **Comments** — add, view, update, delete comments
- **Labels** — view and create labels
- **Members** — list project members for reassignment
- **Cycles** — view sprints/cycles
- **Modules** — view modules
- **Activities** — view issue history/activity log
- **Attachments** — view issue attachments
- **Archive/Reopen** — archive and reopen tickets
- **Defaults** — get current workspace and project settings

## Installation

```bash
git clone <repository>
cd plane-mcp
go build -o plane-mcp .
```

## Connecting to OpenCode

### Configuration

Edit `~/.config/opencode/opencode.json`:

```json
{
  `$schema`: `https://opencode.ai/config.json`,
  `mcp`: {
    `plane-mcp`: {
      `type`: `local`,
      `command`: [`/path/to/plane-mcp`]
    }
  }
}
```

OpenCode passes its own `PLANE_*` variables automatically.

## Environment Variables

### When Running Manually

```bash
PLANE_API_KEY=plane_api_xxx
PLANE_WORKSPACE=my-workspace
PLANE_BASE_URL=http://localhost:8282
PLANE_DEFAULT_PROJECT=My Project
```

### When Running via OpenCode

OpenCode passes its own `PLANE_*` variables automatically.

## Important Notes

### ID Formats

**Project ID** — must be UUID, not state ID or name:

```bash
# Wrong (this is a STATE UUID, not project)
project_id: d0687399-2924-43c1-b745-ea8f80f44406

# Correct (project UUID)
project_id: e427e573-64f1-4998-8f2f-4b0ea77c9337
```

**Issue ID** — can be either:
- **UUID**: `f559c7af-610d-47a4-b50a-944d9ca9fa5c`
- **Sequence ID**: `CATSTHOUGH-6` (auto-resolved via workspace and sequence number)

### Relation Types

Supported issue relation types: `blocking`, `blocked_by`, `duplicate`, `relates_to`, `start_after`, `start_before`, `finish_after`, `finish_before`

## Supported Functions

| Function | Description |
|----------|-------------|
| `get_defaults` | Get configured workspace/project |
| `list_projects` | List all projects in workspace |
| `get_project` | Get project details by UUID |
| `list_issues` | List issues with optional search/filter |
| `get_issue` | Get issue details (includes links, relations, parent) |
| `create_issue` | Create new issue |
| `update_issue` | Update issue (name, desc, state, priority, assignees, labels, dates, parent) |
| `delete_issue` | Delete issue |
| `list_states` | List workflow states |
| `add_comment` | Add comment to issue |
| `list_comments` | List issue comments |
| `update_comment` | Update existing comment |
| `delete_comment` | Delete comment |
| `list_labels` | List project labels |
| `create_label` | Create new label |
| `list_members` | List project members |
| `list_cycles` | List project cycles |
| `get_cycle` | Get cycle details |
| `list_modules` | List project modules |
| `get_module` | Get module details |
| `list_activities` | List issue activity history |
| `list_attachments` | List issue attachments |
| `archive_issue` | Archive issue (soft delete) |
| `reopen_issue` | Reopen archived issue |
| `add_link` | Add URL link to issue |
| `update_link` | Update link title |
| `delete_link` | Delete link |
| `add_relation` | Add relation between issues |
| `delete_relation` | Delete relation |
| `create_attachment` | Add attachment URL to issue |
| `create_cycle` | Create new cycle |
| `create_module` | Create new module |

## API Paths

The server uses Plane API v1 with `work-items` paths:
- `/api/v1/workspaces/{workspace}/projects/{project_id}/work-items/`

## Troubleshooting

### Common Errors

**`workspace 'xxx' not found or no access`**
- Check workspace name spelling (case-sensitive)
- Verify API key has access to this workspace

**`no projects found`**
- Check that project_id is a valid project UUID (not a state ID)
- Use `list_projects` to find correct project UUID

**`issue not found`**
- Issue may have been deleted
- Use `list_issues` to find current issue IDs

## Testing

See [TESTING.md](TESTING.md) for detailed testing instructions.

## Additional Resources

- [EXAMPLES.md](EXAMPLES.md) — API usage examples
- [DEBUG.md](DEBUG.md) — Debug mode guide

## Getting API Key

1. Open Workspace Settings → Members
2. Go to **API Keys** section
3. Create a new key with required permissions

## License

MIT