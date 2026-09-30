package cmd

import "github.com/dimkarp93/install-libs/shellcomplete"

var configFlags = []shellcomplete.Flag{
	{Name: "--config", Files: true},
	{Name: "-y", Bool: true},
	{Name: "--yes", Bool: true},
}

var readOnlyFlags = []shellcomplete.Flag{
	{Name: "--config", Files: true},
}

var migrateFlags = []shellcomplete.Flag{
	{Name: "--config", Files: true},
	{Name: "--from"},
	{Name: "--to"},
}

var completionSpec = shellcomplete.Spec{
	Bin: "kdbx-cli",
	Flags: []shellcomplete.Flag{
		{Name: "--config", Files: true},
		{Name: "--key-store", Files: true},
		{Name: "--secrets"},
		{Name: "--stdin"},
		{Name: "--stdin-keep-open", Bool: true},
		{Name: "--secret-file"},
		{Name: "--template"},
		{Name: "--askpass"},
		{Name: "--dry-run", Bool: true},
		{Name: "--path", Bool: true},
		{Name: "--version", Bool: true},
		{Name: "-v", Bool: true},
		{Name: "--origin", Bool: true},
		{Name: "--buildinfo", Bool: true},
	},
	Args: shellcomplete.Anything,
	Commands: []shellcomplete.Command{
		{Name: "config", Flags: configFlags},
		{Name: "check", Flags: configFlags},
		{Name: "show", Flags: readOnlyFlags},
		{Name: "forget", Flags: readOnlyFlags},
		{Name: "help"},
		{Name: "version"},
	},
}
