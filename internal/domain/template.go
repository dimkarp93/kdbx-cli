package domain

import (
	"regexp"
)

var templatePlaceholderRe = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

var templateWrapperExtensions = []string{".tmpl", ".tpl"}
