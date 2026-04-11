package core

// EventKind identifies one row in the kernel event stream (TUI, logs, headless transports).
type EventKind string

const (
	// KindUserMessage records accepted user text for this turn (after slash handling in the transport).
	KindUserMessage EventKind = "user_message"
	// KindAssistantTextDelta is one streamed content fragment from the model.
	KindAssistantTextDelta EventKind = "assistant_text_delta"
	// KindAssistantFinished closes one assistant generation round (before tool execution, if any).
	KindAssistantFinished EventKind = "assistant_finished"
	// KindModelRefusal is emitted when the model returns a refusal instead of normal output.
	KindModelRefusal EventKind = "model_refusal"
	// KindToolCall is emitted once per tool invocation with a complete id/name/arguments.
	KindToolCall EventKind = "tool_call"
	// KindPermissionPrompt is emitted immediately before blocking on [ConfirmTool] for a dangerous tool.
	KindPermissionPrompt EventKind = "permission_prompt"
	// KindPermissionResult reports whether the user approved the pending dangerous tool.
	KindPermissionResult EventKind = "permission_result"
	// KindToolResult is emitted after a tool result is appended to the transcript (success, decline, or error).
	KindToolResult EventKind = "tool_result"
	// KindError is a non-fatal-to-process error for this turn (stream failure, iteration limit, etc.).
	KindError EventKind = "error"
	// KindTurnComplete is emitted when RunUserTurn returns nil (ready for next user line).
	KindTurnComplete EventKind = "turn_complete"
)

// Event is a single kernel-visible fact. Transports should switch on Kind and read only relevant fields.
type Event struct {
	Kind EventKind `json:"kind"`

	// UserMessage
	UserText string `json:"userText,omitempty"`

	// AssistantTextDelta
	TextChunk string `json:"textChunk,omitempty"`

	// AssistantFinished
	AssistantText   string `json:"assistantText,omitempty"`
	ToolCallCount   int    `json:"toolCallCount,omitempty"`
	FinishReason    string `json:"finishReason,omitempty"`
	AssistantRounds int    `json:"assistantRounds,omitempty"` // 1-based index within this RunUserTurn loop

	// ModelRefusal / Error
	Message string `json:"message,omitempty"`

	// ToolCall
	ToolCallID   string         `json:"toolCallId,omitempty"`
	ToolName     string         `json:"toolName,omitempty"`
	ToolArgs     map[string]any `json:"toolArgs,omitempty"`
	ToolArgsJSON string         `json:"toolArgsJson,omitempty"`

	// PermissionPrompt / PermissionResult (prompt also sets ToolArgs for the pending call).
	PermissionTool   string `json:"permissionTool,omitempty"`
	PermissionReason string `json:"permissionReason,omitempty"` // e.g. dangerous_tool, policy_allow, policy_deny

	// PermissionResult
	PermissionApproved           bool     `json:"permissionApproved,omitempty"`
	PermissionDeclineNote        string   `json:"permissionDeclineNote,omitempty"`
	PermissionRulesAdded         []string `json:"permissionRulesAdded,omitempty"`
	PermissionSessionAutoApprove bool     `json:"permissionSessionAutoApprove,omitempty"`

	// ToolResult (content matches what is sent back to the model)
	ToolResultText string `json:"toolResultText,omitempty"`
	ToolExecError  string `json:"toolExecError,omitempty"`
}

// EventHandler receives kernel events. Implementations must return quickly; do not call RunUserTurn from here.
type EventHandler func(Event)
