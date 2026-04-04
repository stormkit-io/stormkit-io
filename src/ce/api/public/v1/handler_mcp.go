package publicapiv1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// mcpMaxBodySize is the maximum accepted request body size for MCP requests.
// Prevents memory exhaustion from arbitrarily large payloads.
const mcpMaxBodySize = 1 << 20 // 1 MiB

// mcpProtocolVersion is the MCP protocol version this server speaks.
// "2025-11-25" is the Streamable HTTP transport revision — a single POST
// endpoint handles all client→server messages, and an optional GET endpoint
// provides an SSE stream for server→client notifications.
const mcpProtocolVersion = "2025-11-25"

// JSON-RPC 2.0 pre-defined error codes (https://www.jsonrpc.org/specification#error_object).
const (
	rpcErrParse         = -32700
	rpcErrInvalidReq    = -32600
	rpcErrMethodUnknown = -32601
	rpcErrInvalidParams = -32602
	rpcErrInternal      = -32603
)

// jsonRPCRequest is the incoming JSON-RPC 2.0 envelope.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// toolCallParams holds the tools/call parameters.
type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func handlerMCP(req *RequestContext) *shttp.Response {
	rpcReq := &jsonRPCRequest{}

	// Enforce a body size limit to prevent memory exhaustion from oversized payloads.
	req.Request.Body = http.MaxBytesReader(req.Writer(), req.Request.Body, mcpMaxBodySize)

	// Decode the body directly so we can distinguish JSON syntax errors
	// (→ -32700 parse error) from structural/type errors (→ -32600 invalid
	// request), as required by the JSON-RPC 2.0 spec.
	body, err := io.ReadAll(req.Request.Body)

	if err != nil {
		var maxErr *http.MaxBytesError

		if errors.As(err, &maxErr) {
			return jsonRPCError(nil, rpcErrInvalidReq, "request body too large")
		}

		return jsonRPCError(nil, rpcErrInternal, "internal error: failed to read request body")
	}

	// Restore the body for any downstream reads.
	req.Request.Body = io.NopCloser(bytes.NewReader(body))

	if err := json.Unmarshal(body, rpcReq); err != nil {
		switch err.(type) {
		case *json.SyntaxError:
			return jsonRPCError(nil, rpcErrParse, "parse error: "+err.Error())
		default:
			return jsonRPCError(nil, rpcErrInvalidReq, "invalid request: "+err.Error())
		}
	}

	if rpcReq.JSONRPC != "2.0" {
		return jsonRPCError(rpcReq.ID, rpcErrInvalidReq, "invalid request: jsonrpc must be \"2.0\"")
	}

	if rpcReq.Method == "" {
		return jsonRPCError(rpcReq.ID, rpcErrInvalidReq, "invalid request: method is required")
	}

	switch rpcReq.Method {
	case "initialize":
		return jsonRPCResult(rpcReq.ID, map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "stormkit", "version": config.Get().Version.Tag},
		})

	case "notifications/initialized":
		// JSON-RPC notifications carry no id; per JSON-RPC 2.0 the server MUST
		// NOT reply. When an id is present the message is a request, not a
		// notification, so we return an empty result.
		if rpcReq.ID == nil {
			return &shttp.Response{Status: http.StatusAccepted}
		}

		return jsonRPCResult(rpcReq.ID, map[string]any{})

	case "tools/list":
		return jsonRPCResult(rpcReq.ID, map[string]any{
			"tools": mcpAllTools(),
		})

	case "tools/call":
		params := &toolCallParams{}
		if err := json.Unmarshal(rpcReq.Params, params); err != nil {
			return jsonRPCError(rpcReq.ID, rpcErrInvalidParams, "invalid params")
		}

		if params.Arguments == nil {
			params.Arguments = map[string]any{}
		}

		return mcpDispatch(&RequestContextMCP{RequestContext: req}, rpcReq.ID, params)

	default:
		return jsonRPCError(rpcReq.ID, rpcErrMethodUnknown, "method not found: "+rpcReq.Method)
	}
}

// mcpDispatch routes a tools/call to the correct wrapper and converts the
// *shttp.Response into a JSON-RPC result envelope. Tool wrappers return an
// *shttp.Response directly — same as every other handler in this package.
func mcpDispatch(req *RequestContextMCP, id any, params *toolCallParams) *shttp.Response {
	var resp *shttp.Response

	switch params.Name {
	case "deploy":
		resp = mcpDeploy(req, id, params.Arguments)
	case "get_deployment":
		resp = mcpGetDeployment(req, params.Arguments)
	case "publish_deployment":
		resp = mcpPublishDeployment(req, params.Arguments)
	case "list_apps":
		resp = mcpListApps(req, params.Arguments)
	case "list_environments":
		resp = mcpListEnvironments(req, params.Arguments)
	case "create_environment":
		resp = mcpCreateEnvironment(req, id, params.Arguments)
	case "update_environment":
		resp = mcpUpdateEnvironment(req, id, params.Arguments)
	case "list_domains":
		resp = mcpListDomains(req, params.Arguments)
	default:
		return jsonRPCError(id, rpcErrMethodUnknown, "unknown tool: "+params.Name)
	}

	// Map HTTP error responses to MCP isError content so the agent can read
	// the reason while the transport stays HTTP 200 (JSON-RPC convention).
	if resp.Status >= 400 {
		var msg []byte

		if resp.Data != nil {
			msg, _ = json.Marshal(resp.Data)
		} else {
			msg, _ = json.Marshal(map[string]any{
				"status": resp.Status,
				"error":  http.StatusText(resp.Status),
			})
		}

		return jsonRPCResult(id, map[string]any{
			"isError": true,
			"content": []map[string]any{{
				"type": "text",
				"text": string(msg),
			}},
		})
	}

	text, _ := json.Marshal(resp.Data)

	return jsonRPCResult(id, map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(text)}},
	})
}

func jsonRPCResult(id any, result any) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		},
	}
}

func jsonRPCError(id any, code int, message string) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK, // JSON-RPC errors are always HTTP 200
		Data: map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"error": map[string]any{
				"code":    code,
				"message": message,
			},
		},
	}
}

// handlerMCPStream implements the optional GET SSE endpoint of the MCP
// Streamable HTTP transport (2025-11-25). The server has no server-initiated
// messages, so the stream only carries keep-alive comments every 15 s. The
// handler blocks until the client disconnects and returns nil so that shttp
// does not attempt to write an additional response.
func handlerMCPStream(req *RequestContext) *shttp.Response {
	w := req.Writer()
	flusher, ok := w.(http.Flusher)

	if !ok {
		return &shttp.Response{
			Status: http.StatusInternalServerError,
			Data:   map[string]any{"error": "streaming not supported"},
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Write an initial SSE comment to force the response writer chain (e.g.
	// the gzip middleware) to commit the response headers and begin streaming.
	// Without a Write call, GzipResponseWriter.Flush() is a no-op and the
	// client never receives the headers.
	fmt.Fprint(w, ": ping\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-req.Context().Done():
			return nil
		case <-ticker.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}
