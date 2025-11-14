package exampleutil

import (
	"log"

	"github.com/djbozjr/conflata"
)

// ReportWarnings logs configuration issues while allowing examples to proceed.
func ReportWarnings(group *conflata.ErrorGroup) {
	if group == nil {
		return
	}
	for _, fieldErr := range group.Fields() {
		log.Printf("configuration warning: %s", fieldErr.Error())
	}
}
