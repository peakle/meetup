package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

func main() {

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		if strings.HasPrefix(path, "/time") {
			ctx.SetContentType("application/json")
			ctx.SetConnectionClose()

			fmt.Fprint(ctx, fmt.Sprintf(`{"currentDateTime": "%s"}`, time.Now().Format("2006-02-01T15:04-07:00")))
		}
	}

	server := fasthttp.Server{
		Handler:              requestHandler,
		IdleTimeout:          3 * time.Second,
		MaxKeepaliveDuration: 3 * time.Second,
		ReadTimeout:          3 * time.Second,
		WriteTimeout:         3 * time.Second,
		TCPKeepalive:         false,
		DisableKeepalive:     true,
		CloseOnShutdown:      true,
	}

	log.Fatal(server.ListenAndServe(":80"))
}
