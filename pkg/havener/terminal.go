package havener

import (
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

var FixedTerminalWidth = -1

var DefaultTerminalWidth = 80

func GetTerminalWidth() int {
	if FixedTerminalWidth > 0 {
		// Return user preference (explicit overwrite)
		return FixedTerminalWidth

	} else if width, _, err := terminal.GetSize(int(os.Stdout.Fd())); err == nil {
		// Return value read from terminal
		return width

	} else {
		// Return default fall-back value
		return DefaultTerminalWidth
	}
}
