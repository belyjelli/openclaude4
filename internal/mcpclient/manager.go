package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/gitlawb/openclaude4/internal/config"
	"github.com/gitlawb/openclaude4/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Manager holds MCP client sessions started for chat. Call [Manager.Close] on exit.
type Manager struct {
	Servers []ServerTools
}

// ServerTools summarizes one connected MCP server.
type ServerTools struct {
	Name         string
	OpenAINames  []string
	MCPNames     []string
	Session      *mcp.ClientSession
	approval     string
}

// ConnectAndRegister starts stdio MCP servers from cfg, lists tools (with pagination), and registers them on reg.
// Servers that fail to start are skipped; messages are written to log (e.g. os.Stderr).
func ConnectAndRegister(ctx context.Context, reg *tools.Registry, servers []config.MCPServer, log io.Writer) *Manager {
	if len(servers) == 0 {
		return &Manager{}
	}
	cli := mcp.NewClient(&mcp.Implementation{Name: "openclaude", Version: "4"}, &mcp.ClientOptions{
		Capabilities: &mcp.ClientCapabilities{},
	})
	m := &Manager{}

	used := map[string]struct{}{}
	for _, t := range reg.List() {
		used[t.Name()] = struct{}{}
	}

	for _, srv := range servers {
		cmd := exec.CommandContext(ctx, srv.Command[0], srv.Command[1:]...)
		cmd.Env = os.Environ()
		for k, v := range srv.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
		cmd.Stderr = os.Stderr
		if log != nil && log != os.Stderr {
			cmd.Stderr = io.MultiWriter(os.Stderr, log)
		}

		sess, err := cli.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
		if err != nil {
			fmt.Fprintf(OrDiscard(log), "mcp: server %q: connect: %v\n", srv.Name, err)
			continue
		}

		mcpTools, err := listAllTools(ctx, sess)
		if err != nil {
			_ = sess.Close()
			fmt.Fprintf(OrDiscard(log), "mcp: server %q: list tools: %v\n", srv.Name, err)
			continue
		}

		approval := normalizeApproval(srv.Approval)
		danger := approval == "ask"
		var oaiNames, nativeNames []string

		for _, mt := range mcpTools {
			if mt == nil || strings.TrimSpace(mt.Name) == "" {
				continue
			}
			base := OpenAIToolName(srv.Name, mt.Name)
			oai := UniqueOpenAIName(base, used)
			used[oai] = struct{}{}

			desc := strings.TrimSpace(mt.Description)
			reg.Register(&toolAdapter{
				session:    sess,
				openAIName: oai,
				mcpName:    mt.Name,
				desc:       desc,
				params:     InputSchemaToParameters(mt.InputSchema),
				dangerous:  danger,
				serverName: srv.Name,
			})
			oaiNames = append(oaiNames, oai)
			nativeNames = append(nativeNames, mt.Name)
		}

		m.Servers = append(m.Servers, ServerTools{
			Name:        srv.Name,
			OpenAINames: oaiNames,
			MCPNames:    nativeNames,
			Session:     sess,
			approval:    approval,
		})
	}
	return m
}

// OrDiscard returns w or io.Discard if w is nil.
func OrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

func normalizeApproval(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "always", "never":
		return strings.ToLower(strings.TrimSpace(s))
	default:
		return "ask"
	}
}

func listAllTools(ctx context.Context, sess *mcp.ClientSession) ([]*mcp.Tool, error) {
	var out []*mcp.Tool
	cursor := ""
	for {
		res, err := sess.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		out = append(out, res.Tools...)
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}
	return out, nil
}

// Close closes all MCP sessions.
func (m *Manager) Close() {
	if m == nil {
		return
	}
	for _, s := range m.Servers {
		if s.Session != nil {
			_ = s.Session.Close()
		}
	}
}

// DescribeServers returns a short multi-line summary for slash /mcp list.
func (m *Manager) DescribeServers() string {
	if m == nil || len(m.Servers) == 0 {
		return "No MCP servers connected."
	}
	var b strings.Builder
	for _, s := range m.Servers {
		fmt.Fprintf(&b, "- %s: %d tool(s), approval=%s\n", s.Name, len(s.OpenAINames), s.approval)
		for i, oai := range s.OpenAINames {
			mcpN := ""
			if i < len(s.MCPNames) {
				mcpN = s.MCPNames[i]
			}
			fmt.Fprintf(&b, "    %s (MCP %q)\n", oai, mcpN)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

type toolAdapter struct {
	session    *mcp.ClientSession
	openAIName string
	mcpName    string
	desc       string
	params     map[string]any
	dangerous  bool
	serverName string
}

func (t *toolAdapter) Name() string { return t.openAIName }

func (t *toolAdapter) Description() string {
	if t.desc != "" {
		return "[" + t.serverName + " MCP] " + t.desc
	}
	return "[" + t.serverName + " MCP] " + t.mcpName
}

func (t *toolAdapter) Parameters() map[string]any { return t.params }

func (t *toolAdapter) IsDangerous() bool { return t.dangerous }

func (t *toolAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	if args == nil {
		args = map[string]any{}
	}
	res, err := t.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      t.mcpName,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	return formatCallToolResult(res), nil
}

func formatCallToolResult(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		switch x := c.(type) {
		case *mcp.TextContent:
			b.WriteString(x.Text)
		case *mcp.ImageContent:
			fmt.Fprintf(&b, "[image %s, %d bytes]", x.MIMEType, len(x.Data))
		case *mcp.AudioContent:
			fmt.Fprintf(&b, "[audio %s, %d bytes]", x.MIMEType, len(x.Data))
		case *mcp.ResourceLink:
			fmt.Fprintf(&b, "[resource_link uri=%s name=%s]", x.URI, x.Name)
		case *mcp.EmbeddedResource:
			if x.Resource != nil {
				if x.Resource.Text != "" {
					b.WriteString(x.Resource.Text)
				} else if len(x.Resource.Blob) > 0 {
					fmt.Fprintf(&b, "[blob %s, %d bytes]", x.Resource.MIMEType, len(x.Resource.Blob))
				} else {
					b.WriteString("[embedded resource]")
				}
			} else {
				b.WriteString("[embedded resource]")
			}
		default:
			raw, err := json.Marshal(c)
			if err != nil {
				fmt.Fprintf(&b, "%T", c)
			} else {
				b.Write(raw)
			}
		}
	}
	if res.StructuredContent != nil {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("structuredContent: ")
		j, _ := json.Marshal(res.StructuredContent)
		b.Write(j)
	}
	out := strings.TrimSpace(b.String())
	if res.IsError {
		if out == "" {
			return "[tool error]"
		}
		return "[tool error]\n" + out
	}
	if out == "" {
		return "(empty result)"
	}
	return out
}
