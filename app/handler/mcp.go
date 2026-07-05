package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sprungknoedl/dagobert/app/model"
)

// NewMcpHandler builds the read-only MCP endpoint. It only needs the store,
// so it is a standalone http.Handler rather than a set of Handler methods.
func NewMcpHandler(store *model.Store) http.Handler {
	srv := server.NewMCPServer("dagobert", "1.0.0")

	// list_cases is the only tool without a case_id argument.
	srv.AddTool(
		mcp.NewTool("list_cases", mcp.WithDescription("List all investigation cases (id, name, classification, severity, status).")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return jsonResult(store.ListCases())
		},
	)

	// addCaseTool registers a read-only tool taking a required case_id argument and
	// returning the JSON-marshalled result of fn.
	addCaseTool := func(name, description string, fn func(cid string) (any, error)) {
		srv.AddTool(
			mcp.NewTool(name,
				mcp.WithDescription(description),
				mcp.WithString("case_id", mcp.Required(), mcp.Description("The id of the case."))),
			func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				cid, err := req.RequireString("case_id")
				if err != nil {
					return mcp.NewToolResultError(err.Error()), nil
				}
				return jsonResult(fn(cid))
			},
		)
	}

	addCaseTool("get_case", "Get one case with its preloaded contents and counts.",
		func(cid string) (any, error) { return store.GetCaseFull(cid) })
	addCaseTool("list_events", "List the timeline events of a case.",
		func(cid string) (any, error) { return store.ListEvents(cid) })
	addCaseTool("list_assets", "List the assets of a case.",
		func(cid string) (any, error) { return store.ListAssets(cid) })
	addCaseTool("list_indicators", "List the indicators of a case.",
		func(cid string) (any, error) { return store.ListIndicators(cid) })
	addCaseTool("list_malware", "List the malware entries of a case.",
		func(cid string) (any, error) { return store.ListMalware(cid) })
	addCaseTool("list_notes", "List the notes of a case.",
		func(cid string) (any, error) { return store.ListNotes(cid) })
	addCaseTool("list_tasks", "List the tasks of a case.",
		func(cid string) (any, error) { return store.ListTasks(cid) })
	addCaseTool("list_evidences", "List the evidences of a case.",
		func(cid string) (any, error) { return store.ListEvidences(cid) })

	return server.NewStreamableHTTPServer(srv, server.WithStateLess(true))
}

// jsonResult marshals a store result to a JSON text tool result, surfacing any
// store error as an MCP tool error.
func jsonResult(v any, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}
