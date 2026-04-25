import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { CallToolResultSchema } from "@modelcontextprotocol/sdk/types.js";

const PLANE_MCP_PATH = "./plane-mcp";

const WORKSPACE = "catsthoughts";
const PROJECT_ID = "e427e573-64f1-4998-8f2f-4b0ea77c9337";

let client: Client;
let transport: StdioClientTransport;
let testIssueId: string = "";
let testCommentId: string = "";
let testLinkId: string = "";

async function callTool(name: string, args: any = {}): Promise<string> {
  try {
    const result = await client.request(
      { method: "tools/call", params: { name, arguments: args } },
      CallToolResultSchema
    );

    if (!result.content || result.content.length === 0) {
      return "(empty response)";
    }

    return result.content[0].text;
  } catch (err: any) {
    return `ERROR: ${err.message}`;
  }
}

async function callToolParse(name: string, args: any = {}): Promise<any> {
  const text = await callTool(name, args);
  if (text.startsWith("ERROR:")) {
    throw new Error(text);
  }
  return text;
}

async function extractId(text: string, pattern: RegExp): Promise<string | null> {
  const match = text.match(pattern);
  return match ? match[1] : null;
}

async function main() {
  console.log("=== OpenCode E2E Test - Reproducible ===\n");

  transport = new StdioClientTransport({
    command: PLANE_MCP_PATH,
    args: [],
    env: {},
  });

  client = new Client(
    { name: "e2e-test", version: "1.0.0" },
    { capabilities: {} }
  );

  await client.connect(transport);
  console.log("Connected to MCP server\n");

  try {
    // Step 1: Create test issue
    console.log("Creating test issue...");
    let result = await callToolParse("create_issue", {
      workspace: WORKSPACE,
      project_id: PROJECT_ID,
      name: "E2E Test Issue",
      description: "Created by e2e test for validation"
    });
    testIssueId = await extractId(result, /\[([a-f0-9-]{36})\]/);
    if (!testIssueId) throw new Error("Failed to get test issue ID");
    console.log(`Created test issue: ${testIssueId}\n`);

    // Step 2: Test get_issue
    console.log("1. get_issue...");
    result = await callToolParse("get_issue", { workspace: WORKSPACE, issue_id: testIssueId });
    console.log("   OK - issue retrieved\n");

    // Step 3: Test list_states
    console.log("2. list_states...");
    result = await callToolParse("list_states", { workspace: WORKSPACE, project_id: PROJECT_ID });
    console.log("   OK - states listed\n");

    // Step 4: Test list_labels
    console.log("3. list_labels...");
    result = await callToolParse("list_labels", { workspace: WORKSPACE, project_id: PROJECT_ID });
    console.log("   OK - labels listed\n");

    // Step 5: Test list_members
    console.log("4. list_members...");
    result = await callToolParse("list_members", { workspace: WORKSPACE, project_id: PROJECT_ID });
    console.log("   OK - members listed\n");

    // Step 6: Test list_cycles
    console.log("5. list_cycles...");
    result = await callToolParse("list_cycles", { workspace: WORKSPACE, project_id: PROJECT_ID });
    console.log("   OK - cycles listed\n");

    // Step 7: Test list_modules
    console.log("6. list_modules...");
    result = await callToolParse("list_modules", { workspace: WORKSPACE, project_id: PROJECT_ID });
    console.log("   OK - modules listed\n");

    // Step 8: Test update_issue with assignee
    console.log("7. update_issue (assignee)...");
    result = await callToolParse("update_issue", {
      workspace: WORKSPACE,
      issue_id: testIssueId,
      priority: "high"
    });
    console.log("   OK - issue updated\n");

    // Step 9: Test add_comment
    console.log("8. add_comment...");
    result = await callToolParse("add_comment", {
      workspace: WORKSPACE,
      issue_id: testIssueId,
      comment: "Test comment from e2e"
    });
    testCommentId = await extractId(result, /\[([a-f0-9-]{36})\]/);
    console.log(`   OK - comment added: ${testCommentId}\n`);

    // Step 10: Test list_comments
    console.log("9. list_comments...");
    result = await callToolParse("list_comments", { workspace: WORKSPACE, issue_id: testIssueId });
    console.log("   OK - comments listed\n");

    // Step 11: Test update_comment
    if (testCommentId) {
      console.log("10. update_comment...");
      result = await callToolParse("update_comment", {
        workspace: WORKSPACE,
        issue_id: testIssueId,
        comment_id: testCommentId,
        comment: "Updated comment text"
      });
      console.log("   OK - comment updated\n");
    }

    // Step 12: Test add_link
    console.log("11. add_link...");
    result = await callToolParse("add_link", {
      workspace: WORKSPACE,
      issue_id: testIssueId,
      url: "https://example.com",
      title: "Test Link"
    });
    testLinkId = await extractId(result, /\[([a-f0-9-]{36})\]/);
    console.log(`   OK - link added: ${testLinkId}\n`);

    // Step 13: Test list_activities
    console.log("12. list_activities...");
    result = await callToolParse("list_activities", { workspace: WORKSPACE, issue_id: testIssueId });
    console.log("   OK - activities listed\n");

    // Step 14: Test list_attachments
    console.log("13. list_attachments...");
    result = await callToolParse("list_attachments", { workspace: WORKSPACE, issue_id: testIssueId });
    console.log("   OK - attachments listed\n");

    // Step 15: Test create_label
    console.log("14. create_label...");
    result = await callToolParse("create_label", {
      workspace: WORKSPACE,
      project_id: PROJECT_ID,
      name: "e2e-label-" + Date.now(),
      color: "#FF5733"
    });
    console.log("   OK - label created\n");

    // Step 16: Test archive/reopen
    console.log("15. archive_issue...");
    result = await callToolParse("archive_issue", { workspace: WORKSPACE, issue_id: testIssueId });
    console.log("   OK - issue archived\n");

    console.log("16. reopen_issue...");
    result = await callToolParse("reopen_issue", { workspace: WORKSPACE, issue_id: testIssueId });
    console.log("   OK - issue reopened\n");

    // Cleanup
    console.log("\n=== Cleanup ===");

    if (testCommentId) {
      console.log("Deleting test comment...");
      await callTool("delete_comment", {
        workspace: WORKSPACE,
        issue_id: testIssueId,
        comment_id: testCommentId
      });
    }

    console.log("Deleting test issue...");
    result = await callTool("delete_issue", { workspace: WORKSPACE, issue_id: testIssueId });

    console.log("\n=== All tests passed! ===");

  } catch (err) {
    console.error("Error:", err);
    // Cleanup on error
    if (testIssueId) {
      console.log("\nCleaning up test issue...");
      await callTool("delete_issue", { workspace: WORKSPACE, issue_id: testIssueId });
    }
  }

  await client.close();
}

main().catch(console.error);
