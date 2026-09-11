# crawl

Read any web page as clean markdown. Pure Go, no browser needed.

```sh
go install github.com/hanzoai/crawl/cmd/crawl@latest
crawl https://go.dev/doc/effective_go
```

## In your app

```go
page, err := crawl.Fetch(ctx, "https://example.com")
fmt.Println(page.Title, page.Markdown)
```

`crawl.Read` does the same and keeps every page in an archive you bind with
`crawl.Bind(store)` — anything with `Get` and `Put`. A page already read under a
scope is answered without touching the network.

## As a service

```sh
crawl serve 127.0.0.1:8787
curl -s localhost:8787/v1/crawl -d '{"url":"https://example.com"}'
```

The same server is an MCP endpoint at `/mcp` (tool `read_page`) and publishes its
OpenAPI document. `serve.Mount(app, serve.Options{...})` puts the endpoint on your
own [zip](https://github.com/zap-proto/zip) app, with your own admission and
scope. Hanzo Cloud serves it at `https://api.hanzo.ai/v1/crawl`.

`crawl serve` admits every caller. Keep it on loopback, or mount it behind your
own authentication.

## Safe by default

A crawler fetches addresses its callers choose. crawl dials only public unicast
addresses — never loopback, private, link-local or cloud metadata — and checks at
the dialer, so neither DNS rebinding nor a redirect can walk around it.

## Pages that need a browser

A single-page app answers a plain GET with an empty shell. Bind a headless
browser service with `crawl.BindRenderer(crawl.Renderer{URL: ..., Token: ...})`
and crawl asks it for the rendered page whenever the static read comes back too
thin, keeping whichever is richer. `crawl.BindMeter` bills those renders.

## License

Apache-2.0 OR MIT.
