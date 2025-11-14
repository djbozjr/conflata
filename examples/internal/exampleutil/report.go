package exampleutil

import (
	"log"

	"github.com/djbozjr/conflata"
)

// ReportWarnings logs configuration issues when the provided error is an
// *conflata.ErrorGroup. It returns true if warnings were emitted.
func ReportWarnings(err error) bool {
	group, ok := err.(*conflata.ErrorGroup)
	if !ok || group == nil {
		return false
	}
	for _, fieldErr := range group.Fields() {
		log.Printf("configuration warning: %s", fieldErr.Error())
	}
	return true
}
