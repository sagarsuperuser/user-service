package main

import (
	"github.com/sagarsuperuser/userprofile/cmd/runner"
	"github.com/sagarsuperuser/userprofile/server/settings"
)

func main() {
	runner := runner.NewRunner(settings.NewSettings())
	runner.Run()
}
