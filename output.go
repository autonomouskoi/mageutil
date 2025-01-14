package mageutil

import (
	"fmt"

	"github.com/magefile/mage/mg"
)

// VerboseF prints a message to stdout if mage is set to Verbose
func VerboseF(tmpl string, args ...any) {
	if !mg.Verbose() {
		return
	}
	fmt.Printf(tmpl, args...)
}
