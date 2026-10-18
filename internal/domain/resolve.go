package domain

import (
	"github.com/dimkarp93/kdbx-cli/internal/config"
)

type Resolved struct {
	KeyStore      string
	Secrets       map[string]string
	Stdin         []string
	StdinKeepOpen bool
	Files         []string
	Templates     []config.Template
	Askpass       string
	AskpassReads  int
}

type Overrides struct {
	KeyStore      string
	Secrets       map[string]string
	Stdin         []string
	StdinKeepOpen bool
	Files         []string
	Templates     []config.Template
	Askpass       string
	AskpassReads  int
}
