package handler

import (
	"fmt"
	"strings"
)

func buildMyLink(botUsername, tokenValue string) string {
	return fmt.Sprintf("https://t.me/%s?start=%s", botUsername, tokenValue)
}

// commandPayload returns the argument after a "/command" prefix, or "" when
// there is none. It tolerates the "/command@botname" form.
func commandPayload(text string) string {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return ""
	}
	return fields[1]
}
