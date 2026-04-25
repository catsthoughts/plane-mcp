# Plane MCP — Examples

## Available Tools

### get_defaults
Current workspace and project settings.

```json
{ `name`: `get_defaults`, `arguments`: {} }
```

### list_projects
List all projects in workspace.

```json
{ `name`: `list_projects`, `arguments`: { `workspace`: `my-workspace` } }
```

### get_project
Details of a specific project.

```json
{
  `name`: `get_project`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid` }
}
```

### list_issues
List tickets with optional search/filter.

```json
{
  `name`: `list_issues`,
  `arguments`: {
    `workspace`: `my-workspace`,
    `project_id`: `uuid`,
    `search`: `login bug`,
    `state`: `state-uuid`
  }
}
```

### get_issue
Ticket details (includes links, relations, parent).

```json
{
  `name`: `get_issue`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid` }
}
```

Issue ID can be UUID or sequence ID like `CATSTHOUGH-6`.

### create_issue
Create a new ticket.

```json
{
  `name`: `create_issue`,
  `arguments`: {
    `workspace`: `my-workspace`,
    `project_id`: `uuid`,
    `name`: `Ticket title`,
    `description`: `<p>HTML description</p>`,
    `state`: `state-uuid`,
    `priority`: `urgent|high|medium|low|none`
  }
}
```

### update_issue
Update ticket fields.

```json
{
  `name`: `update_issue`,
  `arguments`: {
    `workspace`: `my-workspace`,
    `issue_id`: `uuid`,
    `name`: `New title`,
    `description`: `<p>HTML description</p>`,
    `state`: `state-uuid`,
    `priority`: `urgent|high|medium|low|none`,
    `assignees`: [`user-uuid`],
    `labels`: [`label-uuid`],
    `parent`: `parent-issue-uuid`,
    `start_date`: `2026-04-25`,
    `target_date`: `2026-04-30`
  }
}
```

### delete_issue
Delete a ticket.

```json
{
  `name`: `delete_issue`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid` }
}
```

### list_states
List workflow states.

```json
{
  `name`: `list_states`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid` }
}
```

### add_comment
Add a comment.

```json
{
  `name`: `add_comment`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid`, `comment`: `Text` }
}
```

### list_comments
View comments.

```json
{
  `name`: `list_comments`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid` }
}
```

### update_comment
Update existing comment.

```json
{
  `name`: `update_comment`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid`, `comment_id`: `uuid`, `comment`: `New text` }
}
```

### delete_comment
Delete comment.

```json
{
  `name`: `delete_comment`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid`, `comment_id`: `uuid` }
}
```

### list_labels
View labels.

```json
{
  `name`: `list_labels`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid` }
}
```

### create_label
Create a label.

```json
{
  `name`: `create_label`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid`, `name`: `bug`, `color`: `#FF0000` }
}
```

### list_members
View project members.

```json
{
  `name`: `list_members`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid` }
}
```

### list_cycles
View project cycles.

```json
{
  `name`: `list_cycles`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid` }
}
```

### get_cycle
Get cycle details.

```json
{
  `name`: `get_cycle`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid`, `cycle_id`: `uuid` }
}
```

### list_modules
View project modules.

```json
{
  `name`: `list_modules`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid` }
}
```

### get_module
Get module details.

```json
{
  `name`: `get_module`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid`, `module_id`: `uuid` }
}
```

### list_activities
View issue activity history (all changes).

```json
{
  `name`: `list_activities`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid` }
}
```

### list_attachments
View issue attachments.

```json
{
  `name`: `list_attachments`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid` }
}
```

### add_link
Add URL link to issue.

```json
{
  `name`: `add_link`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid`, `url`: `https://github.com/...`, `title`: `GitHub` }
}
```

### update_link
Update link title.

```json
{
  `name`: `update_link`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid`, `link_id`: `uuid`, `title`: `New Title` }
}
```

### delete_link
Delete link.

```json
{
  `name`: `delete_link`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid`, `link_id`: `uuid` }
}
```

### add_relation
Add relation between issues.

```json
{
  `name`: `add_relation`,
  `arguments`: {
    `workspace`: `my-workspace`,
    `issue_id`: `uuid`,
    `target_issue_id`: `uuid`,
    `relation_type`: `blocking|blocked_by|duplicate|relates_to|start_after|start_before|finish_after|finish_before`
  }
}
```

### delete_relation
Delete relation.

```json
{
  `name`: `delete_relation`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid`, `relation_id`: `uuid` }
}
```

### create_attachment
Add attachment URL to issue.

```json
{
  `name`: `create_attachment`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid`, `url`: `https://...`, `title`: `File.pdf` }
}
```

### create_cycle
Create new cycle.

```json
{
  `name`: `create_cycle`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid`, `name`: `Sprint 1` }
}
```

### create_module
Create new module.

```json
{
  `name`: `create_module`,
  `arguments`: { `workspace`: `my-workspace`, `project_id`: `uuid`, `name`: `Backend Module` }
}
```

### archive_issue
Archive ticket (soft delete).

```json
{
  `name`: `archive_issue`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid` }
}
```

### reopen_issue
Reopen archived ticket.

```json
{
  `name`: `reopen_issue`,
  `arguments`: { `workspace`: `my-workspace`, `issue_id`: `uuid` }
}
```

## Natural Language Usage

```
Show projects. use plane-mcp
List tickets. use plane-mcp
Show issue states. use plane-mcp
Update issue with state 'In Progress'. use plane-mcp
Create a new issue 'Fix login bug' with high priority. use plane-mcp
Add comment 'Fixed in v2.0' to the issue. use plane-mcp
Show issue activity. use plane-mcp
List project labels. use plane-mcp
```

## Issue ID Formats

Both UUID and sequence ID formats are supported:
- UUID: `f559c7af-610d-47a4-b50a-944d9ca9fa5c`
- Sequence: `CATSTHOUGH-6` (workspace-project-number)

## Relation Types

Issue relations include: `blocking`, `blocked_by`, `duplicate`, `relates_to`, `start_after`, `start_before`, `finish_after`, `finish_before`