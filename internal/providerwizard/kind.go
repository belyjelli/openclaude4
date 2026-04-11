package providerwizard

// StepKind classifies the current wizard step for UIs (REPL vs TUI).
type StepKind int

const (
	StepMenu StepKind = iota
	StepText
	StepDone
)
