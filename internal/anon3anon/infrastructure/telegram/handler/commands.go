package handler

const (
	CommandStart  = "start"
	CommandHelp   = "help"
	CommandMyLink = "mylink"
	CommandRevoke = "revoke"
	CommandBlock  = "block"
	CommandStop   = "stop"
	CommandDelete = "delete"
)

func CommandNames() []string {
	return []string{
		CommandStart,
		CommandHelp,
		CommandMyLink,
		CommandRevoke,
		CommandBlock,
		CommandStop,
		CommandDelete,
	}
}
