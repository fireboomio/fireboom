package main

import (
	// Make sure to import swag docs, don't remove
	_ "fireboom-server/app/docs"
	"fireboom-server/cmd"
)

var (
	FbVersion string
	FbCommit  string
)

func main() {
	cmd.Execute(FbVersion, FbCommit)
}
