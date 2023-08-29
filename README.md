# Remilia

<img alt="GitHub license" src="https://img.shields.io/badge/license-MIT-blue.svg">

## Description

The Remilia package is a flexible and easy-to-use web crawling library written in Go. The package is designed with concurrency and middleware support to enable a wide variety of web scraping applications.

**⚠️ This repository is a work in progress, so features and APIs are subject to change.**

## Features

- **Concurrent Crawling**: Control the number of concurrent HTTP requests.
- **Middleware Chain**: Define a sequence of URL Generators and HTML Processors to build a flexible crawling pipeline.
- **Limit Rules**: Configure domain restrictions and crawl delays.
- **Customizable Logging**: Uses the `logger` package for fine-grained logging.
- **Data Consumers**: Allows the integration of custom data consumers for additional processing or storage.
- **Resilient**: Error handling at each stage of the pipeline.

## Installation

To install `Remilia`, use the following command:

```bash
go get github.com/ShroXd/remilia
```

## Usage

Here's a simple example to get you started:

```go
package main

import (
    "github.com/ShroXd/remilia"
    "github.com/PuerkitoBio/goquery"
    "net/url"
)

func main() {
    // Create a new Remilia instance
    r := remilia.New("https://example.com")

    // Configure URL generation step for middleware
    r.UseURL("a", func(s *goquery.Selection) *url.URL {
        // Your URL generation logic
    })
    // Configure HTML processing step for middleware
    .UseHTML("div.article", func(s *goquery.Selection) interface{} {
        // Your HTML processing logic
    }, yourDataConsumer)
    // Add the configured middleware to the chain
    .AddToChain()

    // Start the crawling
    err := r.Start()
    if err != nil {
        // Handle error
    }
}
```

## API Reference

- `New(url string, options ...Option) *Remilia`: Create a new Remilia instance.
- `UseURL(selector string, urlGenFn func(s *goquery.Selection) *url.URL) *Remilia`: Define a middleware for URL generation.
- `UseHTML(selector string, htmlProcFn func(s *goquery.Selection) interface{}, dataConsumer DataConsumer) *Remilia`: Define a middleware for HTML processing.
- `AddToChain() *Remilia`: Adds the current middleware to the middleware chain.
- `Start() error`: Starts the crawling process.

## Examples

For more complex usage, see [Examples](./examples).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE.md) file for details.
