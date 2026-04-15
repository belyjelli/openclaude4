package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/gitlawb/openclaude4/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Manager holds MCP client sessions started for chat. Call [Manager.Close] on exit.
type Manager struct {
	Servers []ServerTools
}

// MCPResource is a listed resource from one MCP server (snapshot at connect).
type MCPResource struct {
	Server string
	URI    string
	Name   string
	Title  string
}

// ServerTools summarizes one connected MCP server.
type ServerTools struct {
	Name        string
	OpenAINames []string
	MCPNames    []string
	Resources   []MCPResource
	Session     *mcp.ClientSession
	approval    string
}

// Approval returns the normalized approval mode: ask, always, or never.
func (s ServerTools) Approval() string { return s.approval }

// ConnectAndRegister starts stdio MCP servers from cfg, lists tools (with pagination), and registers them on reg.
// Servers that fail to start are skipped; messages are written to log (e.g. os.Stderr).
func ConnectAndRegister(ctx context.Context, reg *tools.Registry, servers []ServerConfig, log io.Writer) *Manager {
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
		if len(srv.Command) == 0 {
			continue
		}
		tport := strings.ToLower(strings.TrimSpace(srv.Transport))
		if tport != "" && tport != "stdio" {
			_, _ = fmt.Fprintf(OrDiscard(log), "mcp: server %q: transport %q is not supported yet (only stdio)\n", srv.Name, srv.Transport)
			continue
		}

		cmd := exec.CommandContext(ctx, srv.Command[0], srv.Command[1:]...)
		cmd.Env = os.Environ()
		for k, v := range srv.Env {
			cmd.Env = append(cmd.Env, k+"="+expandEnvVal(v))
		}
		cmd.Stderr = os.Stderr
		if log != nil && log != os.Stderr {
			cmd.Stderr = io.MultiWriter(os.Stderr, log)
		}

		sess, err := cli.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
		if err != nil {
			_, _ = fmt.Fprintf(OrDiscard(log), "mcp: server %q: connect: %v\n", srv.Name, err)
			continue
		}

		mcpTools, err := listAllTools(ctx, sess)
		if err != nil {
			_ = sess.Close()
			_, _ = fmt.Fprintf(OrDiscard(log), "mcp: server %q: list tools: %v\n", srv.Name, err)
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

		var resList []MCPResource
		if serverSupportsResources(sess) {
			mcpRes, err := listAllResources(ctx, sess)
			if err != nil {
				_, _ = fmt.Fprintf(OrDiscard(log), "mcp: server %q: list resources: %v\n", srv.Name, err)
			} else {
				for _, r := range mcpRes {
					if r == nil || strings.TrimSpace(r.URI) == "" {
						continue
					}
					resList = append(resList, MCPResource{
						Server: srv.Name,
						URI:    strings.TrimSpace(r.URI),
						Name:   strings.TrimSpace(r.Name),
						Title:  strings.TrimSpace(r.Title),
					})
				}
			}
		}

		m.Servers = append(m.Servers, ServerTools{
			Name:        srv.Name,
			OpenAINames: oaiNames,
			MCPNames:    nativeNames,
			Resources:   resList,
			Session:     sess,
			approval:    approval,
		})
	}
	return m
}

func expandEnvVal(s string) string {
	return os.Expand(s, func(key string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		return os.Getenv(key)
	})
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

func serverSupportsResources(sess *mcp.ClientSession) bool {
	if sess == nil {
		return false
	}
	init := sess.InitializeResult()
	if init == nil || init.Capabilities == nil || init.Capabilities.Resources == nil {
		return false
	}
	return true
}

func listAllResources(ctx context.Context, sess *mcp.ClientSession) ([]*mcp.Resource, error) {
	var out []*mcp.Resource
	cursor := ""
	for {
		res, err := sess.ListResources(ctx, &mcp.ListResourcesParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		out = append(out, res.Resources...)
		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}
	return out, nil
}

// ReadResourceText loads a resource by URI, preferring the server named serverName when it lists that URI.
func (m *Manager) ReadResourceText(ctx context.Context, serverName, uri string) (string, error) {
	if m == nil {
		return "", fmt.Errorf("mcp: manager is nil")
	}
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return "", fmt.Errorf("mcp: empty resource uri")
	}
	serverName = strings.TrimSpace(serverName)

	var sess *mcp.ClientSession
	if serverName != "" {
		for _, s := range m.Servers {
			if !strings.EqualFold(strings.TrimSpace(s.Name), serverName) {
				continue
			}
			for _, r := range s.Resources {
				if strings.TrimSpace(r.URI) == uri {
					sess = s.Session
					break
				}
			}
			if sess != nil {
				break
			}
		}
	}
	if sess == nil {
		for _, s := range m.Servers {
			for _, r := range s.Resources {
				if strings.TrimSpace(r.URI) == uri {
					sess = s.Session
					break
				}
			}
			if sess != nil {
				break
			}
		}
	}
	if sess == nil {
		return "", fmt.Errorf("mcp: no connected resource with uri %q", uri)
	}

	res, err := sess.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		return "", err
	}
	if res == nil || len(res.Contents) == 0 {
		return "", fmt.Errorf("mcp: empty resource contents for %q", uri)
	}
	var b strings.Builder
	for _, c := range res.Contents {
		if c == nil {
			continue
		}
		if strings.TrimSpace(c.Text) != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(c.Text)
			continue
		}
		if len(c.Blob) > 0 {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			mime := strings.TrimSpace(c.MIMEType)
			if mime == "" {
				mime = "application/octet-stream"
			}
			_, _ = fmt.Fprintf(&b, "[binary resource %s, %d bytes]", mime, len(c.Blob))
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "(empty resource)", nil
	}
	return out, nil
}

// ResourceSuggestCandidates returns resources whose URI, Name, or Title has a case-insensitive
// prefix match to query. Empty query matches all. Results are deduped by URI and sorted by URI.
func (m *Manager) ResourceSuggestCandidates(query string) []MCPResource {
	if m == nil {
		return nil
	}
	q := strings.TrimSpace(query)
	lowQ := strings.ToLower(q)
	var flat []MCPResource
	seen := map[string]struct{}{}
	for _, s := range m.Servers {
		for _, r := range s.Resources {
			uri := strings.TrimSpace(r.URI)
			if uri == "" {
				continue
			}
			if _, ok := seen[uri]; ok {
				continue
			}
			if q != "" {
				lowURI := strings.ToLower(uri)
				lowName := strings.ToLower(strings.TrimSpace(r.Name))
				lowTitle := strings.ToLower(strings.TrimSpace(r.Title))
				if !strings.HasPrefix(lowURI, lowQ) && !strings.HasPrefix(lowName, lowQ) && !strings.HasPrefix(lowTitle, lowQ) {
					continue
				}
			}
			seen[uri] = struct{}{}
			srvName := strings.TrimSpace(r.Server)
			if srvName == "" {
				srvName = s.Name
			}
			flat = append(flat, MCPResource{
				Server: srvName,
				URI:    uri,
				Name:   r.Name,
				Title:  r.Title,
			})
		}
	}
	sort.Slice(flat, func(i, j int) bool {
		return strings.ToLower(flat[i].URI) < strings.ToLower(flat[j].URI)
	})
	return flat
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
		_, _ = fmt.Fprintf(&b, "- %s: %d tool(s), approval=%s\n", s.Name, len(s.OpenAINames), s.approval)
		for i, oai := range s.OpenAINames {
			mcpN := ""
			if i < len(s.MCPNames) {
				mcpN = s.MCPNames[i]
			}
			_, _ = fmt.Fprintf(&b, "    %s (MCP %q)\n", oai, mcpN)
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
			_, _ = fmt.Fprintf(&b, "[image %s, %d bytes]", x.MIMEType, len(x.Data))
		case *mcp.AudioContent:
			_, _ = fmt.Fprintf(&b, "[audio %s, %d bytes]", x.MIMEType, len(x.Data))
		case *mcp.ResourceLink:
			_, _ = fmt.Fprintf(&b, "[resource_link uri=%s name=%s]", x.URI, x.Name)
		case *mcp.EmbeddedResource:
			if x.Resource != nil {
				if x.Resource.Text != "" {
					b.WriteString(x.Resource.Text)
				} else if len(x.Resource.Blob) > 0 {
					_, _ = fmt.Fprintf(&b, "[blob %s, %d bytes]", x.Resource.MIMEType, len(x.Resource.Blob))
				} else {
					b.WriteString("[embedded resource]")
				}
			} else {
				b.WriteString("[embedded resource]")
			}
		default:
			raw, err := json.Marshal(c)
			if err != nil {
				_, _ = fmt.Fprintf(&b, "%T", c)
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
