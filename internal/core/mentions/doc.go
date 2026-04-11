// Package mentions expands v3-style @-references in user prompts before the model runs.
//
// Supported:
//   - @file, @"path with spaces", @file#Ln and @file#Ln-m (workspace-relative or in-workspace absolute paths)
//   - @server:resourceURI for MCP resources when a manager is provided
//
// Not supported (by design in v4):
//   - Teammate / swarm @name (no message bus)
//   - @agent-<type> is left in the text unchanged (no AgentDefinition catalog); use the Task tool or slash commands instead.
//   - @mcp:… tab-completion URIs are skills/MCP UI; free-text @server:uri uses MCP resource read when listed.

package mentions
