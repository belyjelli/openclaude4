package core

// SlashSubmitUser is returned from the chat slash handler when a slash command should
// continue as a normal user message to the model (plain REPL and TUI).
type SlashSubmitUser struct {
	UserText string
}

func (e SlashSubmitUser) Error() string {
	return "slash: deferred user turn"
}

// SlashStartProviderWizard is returned from the slash handler when the TUI should open
// the interactive provider wizard overlay (plain REPL uses stdin instead).
type SlashStartProviderWizard struct{}

func (SlashStartProviderWizard) Error() string { return "slash: start provider wizard" }
