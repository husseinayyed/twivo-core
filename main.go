package main

import (
	"context"

	"github.com/husseinayyed/twivo-core/pkg/bootstrap"
	"github.com/husseinayyed/twivo-core/pkg/twivo"
)

func main() {
	bootstrap.MustRun()
	err := twivo.SetCache(context.Background(), "test", "value", 60)
	if err != nil {
		panic(err)
	}
	val, err := twivo.GetCache(context.Background(), "test")
	if err != nil {
		panic(err)
	}
	println(val.(string))	
	for {

	}
}
