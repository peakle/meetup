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
			fmt.Fprint(ctx, fmt.Sprintf(`{"currentDateTime": "%s"}`, time.Now().Format("2006-02-01T15:04-07:00")))
		} else {
			ctx.SetConnectionClose()
		}
	}

	server := fasthttp.Server{
		Handler:              requestHandler,
		IdleTimeout:          30 * time.Second,
		TCPKeepalivePeriod:   3 * time.Second,
		TCPKeepalive:         true,
		MaxKeepaliveDuration: 30 * time.Second,
		ReadTimeout:          3 * time.Second,
		WriteTimeout:         3 * time.Second,
	}

	log.Fatal(server.ListenAndServe(":80"))
}
