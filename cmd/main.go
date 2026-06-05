package main

import (
	"chat2api/app"
	"context"
)

func main() {
	ctx := context.Background()
	app.Run(ctx)
}
