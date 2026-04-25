# Testing

## Go Tests

```bash
go test . -v
```

## Manual Test

```bash
echo '{"jsonrpc":"2.0",id":1,"method":"tools/list"}' | ./plane-mcp
```

## E2E Tests

Requires Node.js and npm dependencies:

```bash
npm install
npx tsx e2e-test.ts
```

E2E tests create real issues in your Plane instance and clean up after themselves.

### Prerequisites

- Plane API key with appropriate permissions
- A test project to run tests against

### Configuration

Edit the constants at the top of `e2e-test.ts`:

```typescript
const WORKSPACE = "my-workspace";
const PROJECT_ID = "project-uuid";
```

### What E2E Tests Cover

1. **create_issue** — creating new issues
2. **get_issue** — retrieving issue details
3. **list_states** — listing workflow states
4. **list_labels** — listing project labels
5. **list_members** — listing project members
6. **list_cycles** — listing project cycles
7. **list_modules** — listing project modules
8. **update_issue** — updating issue fields
9. **add_comment** — adding comments
10. **list_comments** — viewing comments
11. **update_comment** — updating comments
12. **add_link** — adding links
13. **list_activities** — viewing activity log
14. **list_attachments** — listing attachments
15. **create_label** — creating labels
16. **archive_issue** — archiving issues
17. **reopen_issue** — reopening archived issues
