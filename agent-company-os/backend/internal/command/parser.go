package command

import "strings"

type Command struct {
	Name    string
	Args    []string
	RawText string
	ChatID  int64
	UserID  int64
}

func Parse(raw string, chatID, userID int64) Command {
	fields := strings.Fields(strings.TrimSpace(raw))
	cmd := Command{RawText: raw, ChatID: chatID, UserID: userID}
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		cmd.Name = "unknown"
		return cmd
	}
	name := strings.TrimPrefix(fields[0], "/")
	if at := strings.IndexByte(name, '@'); at >= 0 {
		name = name[:at]
	}
	cmd.Name = strings.ToLower(name)
	if len(fields) > 1 {
		cmd.Args = fields[1:]
	}
	return cmd
}
