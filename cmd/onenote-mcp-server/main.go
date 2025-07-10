package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/gebl/onenote-mcp-server/internal/auth"
	"github.com/gebl/onenote-mcp-server/internal/config"
	"github.com/gebl/onenote-mcp-server/internal/graph"
)

func main() {
	// Set up logging to file if MCP_LOG_FILE env var is set
	logFilePath := os.Getenv("MCP_LOG_FILE")
	if logFilePath != "" {
		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v\n", logFilePath, err)
			os.Exit(1)
		}
		log.SetOutput(f)
		defer f.Close()
	}

	// Mode flag: stdio (default), http, or sse
	mode := flag.String("mode", "stdio", "Serve mode: stdio, http, or sse")
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	tokenPath := "tokens.json"
	tm, err := auth.LoadTokens(tokenPath)
	if err != nil || tm == nil || tm.IsExpired() {
		log.Println("No valid token found or token expired. Starting PKCE flow...")
		// Set up PKCE
		codeVerifier, err := auth.GenerateCodeVerifier()
		if err != nil {
			log.Fatalf("Failed to generate code verifier: %v", err)
		}
		codeChallenge := auth.CodeChallengeS256(codeVerifier)
		state := "mcp-onenote-state" // could randomize for extra security
		redirectPath := "/callback"
		codeCh := make(chan string)
		// Start local server to receive code
		server, err := auth.StartLocalServer(redirectPath, codeCh, state)
		if err != nil {
			log.Fatalf("Failed to start local server: %v", err)
		}
		defer server.Close()
		// Print auth URL for user
		oauthCfg := auth.NewOAuth2Config(cfg.ClientID, cfg.TenantID, cfg.RedirectURI)
		authURL := oauthCfg.GetAuthURL(state, codeChallenge)
		log.Printf("\nPlease visit the following URL in your browser to authenticate:\n%s\n\n", authURL)
		// Wait for code
		code := <-codeCh
		// Exchange code for tokens
		tm, err = oauthCfg.ExchangeCode(context.Background(), code, codeVerifier)
		if err != nil {
			log.Fatalf("Failed to exchange code for tokens: %v", err)
		}
		// Save tokens
		if err := tm.SaveTokens(tokenPath); err != nil {
			log.Printf("Warning: failed to save tokens: %v", err)
		}
		log.Println("Authentication complete. Tokens saved.")
	}

	graphClient := graph.NewClient(tm.AccessToken)
	if err != nil {
		log.Fatalf("Failed to create Graph client: %v", err)
	}

	hooks := &server.Hooks{}

	//func(ctx context.Context, id any, method mcp.MCPMethod, message any)
	hooks.AddOnSuccess(server.OnSuccessHookFunc(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
		jsonData, err := json.Marshal(result)
		if err != nil {
			log.Printf("[hook] Outgoing response (failed to marshal as JSON): %v", err)
			return
		}
		log.Printf("[hook] Outgoing response (JSON): %s", string(jsonData))
	}))

	hooks.AddBeforeAny(server.BeforeAnyHookFunc(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		jsonData, err := json.Marshal(message)
		if err != nil {
			log.Printf("[hook] Incoming request (failed to marshal as JSON): %v", err)
			return
		}
		log.Printf("[hook] Incoming request (JSON): %s", string(jsonData))
	}))

	s := server.NewMCPServer(
		"OneNote MCP Server",
		"1.0.0",
		server.WithHooks(hooks),
	)

	// listNotebooks tool
	listNotebooksTool := mcp.NewTool(
		"listNotebooks",
		mcp.WithDescription("List all OneNote notebooks"),
	)
	s.AddTool(listNotebooksTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		notebooks, err := graphClient.ListNotebooks()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list notebooks: %v", err)), nil
		}
		return mcp.NewToolResultText(stringifyNotebooks(notebooks)), nil
	})

	// listSections tool
	listSectionsTool := mcp.NewTool(
		"listSections",
		mcp.WithDescription("List all sections in a notebook"),
		mcp.WithString("notebookId", mcp.Required(), mcp.Description("Notebook ID to list sections for")),
	)
	s.AddTool(listSectionsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		notebookId, err := req.RequireString("notebookId")
		if err != nil {
			return mcp.NewToolResultError("notebookId is required"), nil
		}
		sections, err := graphClient.ListSections(notebookId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list sections: %v", err)), nil
		}
		return mcp.NewToolResultText(stringifySections(sections)), nil
	})

	// listPages tool
	listPagesTool := mcp.NewTool(
		"listPages",
		mcp.WithDescription("List all pages in a section"),
		mcp.WithString("sectionId", mcp.Required(), mcp.Description("Section ID to list pages for")),
	)
	s.AddTool(listPagesTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sectionId, err := req.RequireString("sectionId")
		if err != nil {
			return mcp.NewToolResultError("sectionId is required"), nil
		}
		pages, err := graphClient.ListPages(sectionId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list pages: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("%v", pages)), nil
	})

	// createPage tool
	createPageTool := mcp.NewTool(
		"createPage",
		mcp.WithDescription("Create a new page in a section"),
		mcp.WithString("sectionId", mcp.Required(), mcp.Description("Section ID to create page in")),
		mcp.WithString("title", mcp.Required(), mcp.Description("Page title")),
		mcp.WithString("content", mcp.Required(), mcp.Description("HTML content for the page")),
	)
	s.AddTool(createPageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sectionId, err := req.RequireString("sectionId")
		if err != nil {
			return mcp.NewToolResultError("sectionId is required"), nil
		}
		title, err := req.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError("title is required"), nil
		}
		content, err := req.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError("content is required"), nil
		}
		result, err := graphClient.CreatePage(sectionId, title, content)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create page: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("%v", result)), nil
	})

	// updatePageContent tool
	updatePageContentTool := mcp.NewTool(
		"updatePageContent",
		mcp.WithDescription("Update the HTML content of a page"),
		mcp.WithString("pageId", mcp.Required(), mcp.Description("Page ID to update")),
		mcp.WithString("content", mcp.Required(), mcp.Description("New HTML content for the page")),
	)
	s.AddTool(updatePageContentTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pageId, err := req.RequireString("pageId")
		if err != nil {
			return mcp.NewToolResultError("pageId is required"), nil
		}
		content, err := req.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError("content is required"), nil
		}
		err = graphClient.UpdatePageContent(pageId, content)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update page content: %v", err)), nil
		}
		return mcp.NewToolResultText("Page content updated successfully"), nil
	})

	// deletePage tool
	deletePageTool := mcp.NewTool(
		"deletePage",
		mcp.WithDescription("Delete a page by ID"),
		mcp.WithString("pageId", mcp.Required(), mcp.Description("Page ID to delete")),
	)
	s.AddTool(deletePageTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pageId, err := req.RequireString("pageId")
		if err != nil {
			return mcp.NewToolResultError("pageId is required"), nil
		}
		err = graphClient.DeletePage(pageId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete page: %v", err)), nil
		}
		return mcp.NewToolResultText("Page deleted successfully"), nil
	})

	// getPageContent tool
	getPageContentTool := mcp.NewTool(
		"getPageContent",
		mcp.WithDescription("Get the HTML content of a page by ID"),
		mcp.WithString("pageId", mcp.Required(), mcp.Description("Page ID to fetch content for")),
	)
	s.AddTool(getPageContentTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pageId, err := req.RequireString("pageId")
		if err != nil {
			return mcp.NewToolResultError("pageId is required"), nil
		}
		content, err := graphClient.GetPageContent(pageId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get page content: %v", err)), nil
		}
		return mcp.NewToolResultText(content), nil
	})

	// getResource tool
	getResourceTool := mcp.NewTool(
		"getResource",
		mcp.WithDescription("Get a OneNote resource (e.g., image) by resource ID, returns base64-encoded data or JSON with filename and data if asBinary is requested"),
		mcp.WithString("resourceId", mcp.Required(), mcp.Description("Resource ID to fetch")),
		mcp.WithString("asBinary", mcp.Description("Return as JSON with filename and base64 data (set to 'true')")),
		mcp.WithString("filename", mcp.Description("Custom filename for binary download (only used if asBinary is true)")),
	)
	s.AddTool(getResourceTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceId := req.GetString("resourceId", "")
		asBinaryStr := req.GetString("asBinary", "")
		filename := req.GetString("filename", "")
		asBinary := asBinaryStr == "true" || asBinaryStr == "1"
		if asBinary {
			data, contentType, err := graphClient.GetResourceBinary(resourceId)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get resource (binary): %v", err)), nil
			}
			if filename == "" {
				filename = resourceId + ".bin"
				if strings.HasPrefix(contentType, "image/") {
					ext := strings.TrimPrefix(contentType, "image/")
					filename = resourceId + "." + ext
				}
			}
			encoded := base64.StdEncoding.EncodeToString(data)
			result := map[string]string{
				"filename": filename,
				"data":     encoded,
			}
			jsonBytes, _ := json.Marshal(result)
			return mcp.NewToolResultText(string(jsonBytes)), nil
		} else {
			data, err := graphClient.GetResource(resourceId)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get resource: %v", err)), nil
			}
			return mcp.NewToolResultText(data), nil
		}
	})

	// listResources tool
	listResourcesTool := mcp.NewTool(
		"listResources",
		mcp.WithDescription("List all OneNote resources (images, files, etc.) for the user"),
	)
	s.AddTool(listResourcesTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resources, err := graphClient.ListResources()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list resources: %v", err)), nil
		}
		jsonBytes, err := json.Marshal(resources)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal resources: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	// getResourceDetails tool (renamed from getResourceMetadata)
	getResourceDetailsTool := mcp.NewTool(
		"getResourceDetails",
		mcp.WithDescription("Get details for a specific OneNote resource by resource ID"),
		mcp.WithString("resourceId", mcp.Required(), mcp.Description("Resource ID to fetch details for")),
	)
	s.AddTool(getResourceDetailsTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resourceId, err := req.RequireString("resourceId")
		if err != nil {
			return mcp.NewToolResultError("resourceId is required"), nil
		}
		meta, err := graphClient.GetResourceMetadata(resourceId)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get resource details: %v", err)), nil
		}
		jsonBytes, err := json.Marshal(meta)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal details: %v", err)), nil
		}
		return mcp.NewToolResultText(string(jsonBytes)), nil
	})

	if *mode == "sse" {
		log.Println("Serving MCP over SSE on :8081 at /mcp")
		sseServer := server.NewSSEServer(s,
			server.WithBaseURL("http://localhost:8081"),
			server.WithStaticBasePath("/mcp"),
		)
		if err := http.ListenAndServe(":8081", sseServer); err != nil {
			log.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		log.Println("Serving MCP over stdio")
		if err := server.ServeStdio(s); err != nil {
			log.Printf("Server error: %v\n", err)
			os.Exit(1)
		}
	}
}

// Helper to format notebooks as JSON or text (replace with your preferred output)
func stringifyNotebooks(notebooks interface{}) string {
	// TODO: marshal to JSON or format as needed
	return fmt.Sprintf("%v", notebooks)
}

func stringifySections(sections interface{}) string {
	// TODO: marshal to JSON or format as needed
	return fmt.Sprintf("%v", sections)
}
