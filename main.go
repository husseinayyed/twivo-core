package main

import (
	"github.com/husseinayyed/twivo-core/pkg/bootstrap"
	"github.com/husseinayyed/twivo-core/pkg/twivo"
)
type Test struct {
	// Add any configuration fields you need here
	username string
	password string
}
func main() {
	bootstrap.MustRun()
	twivo.Core.NewKey[*Test]("exampleKey", 100, 0, 0)
	for {

	}
}
