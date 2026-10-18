package domain

import (
	"regexp"
)

type Mapping struct {
	Name string
	Env  string
}

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const (
	ChannelStdin    = "(stdin)"
	ChannelFile     = "(file)"
	ChannelTemplate = "(template)"
	ChannelAskpass  = "(askpass)"
)
