package core

// SlashSubmitUser is returned from the chat slash handler when a slash command should
// continue as a normal user message to the model (plain REPL and TUI).
type SlashSubmitUser struct {
	UserText string
}

func (e SlashSubmitUser) Error() string {
	return "slash: deferred user turn"
}
