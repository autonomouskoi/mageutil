package mageutil

import (
	"fmt"

	"github.com/magefile/mage/mg"
)

func VerboseF(tmpl string, args ...any) {
	if !mg.Verbose() {
		return
	}
	fmt.Printf(tmpl, args...)
}
