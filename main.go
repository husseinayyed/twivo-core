package main

import (
	"context"

	"github.com/husseinayyed/twivo-core/pkg/bootstrap"
	"github.com/husseinayyed/twivo-core/pkg/twivo"
)

func main() {
	bootstrap.MustRun()
	ctx := context.Background()
	err := twivo.SetCache(ctx, "test", "value", 60)
	if err != nil {
		panic(err)
	}
	val, found, err := twivo.GetCache(ctx, "test")
	if err != nil {
		panic(err)
	}
	if found {
		println(val.(string))
	}
	for {

	}
}
