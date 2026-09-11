// Command crawl reads web pages as markdown, or serves the crawl API.
//
//	crawl <url>...              print each page as markdown
//	crawl serve [host:port]     serve POST /v1/crawl, MCP at /mcp and the OpenAPI document
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/hanzoai/crawl"
	"github.com/hanzoai/crawl/serve"
	"github.com/zap-proto/zip"
)

const usage = `usage:
  crawl <url>...              print each page as markdown
  crawl serve [host:port]     serve POST /v1/crawl (default 127.0.0.1:8787)`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "crawl:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Println(usage)
		return nil
	}
	if args[0] == "serve" {
		addr := "127.0.0.1:8787"
		if len(args) > 1 {
			addr = args[1]
		}
		app := zip.New(zip.Config{AppName: "crawl", DisableStartupMessage: true})
		if err := serve.Mount(app, serve.Options{}); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "crawl: http://%s%s\n", addr, serve.Path)
		return app.Listen("http://" + addr)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	var failed error
	for _, u := range args {
		p, err := crawl.Read(ctx, crawl.Scope{}, u)
		if err != nil {
			failed = errors.Join(failed, err)
			continue
		}
		fmt.Println(p.Markdown)
	}
	return failed
}
