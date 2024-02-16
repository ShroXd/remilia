# Remilia

<img alt="GitHub license" src="https://img.shields.io/badge/license-MIT-blue.svg">

Remilia is a high-performance web scraping framework designed for efficiency. It enables users to concentrate on extracting and utilizing web content, delegating the complexity of web scraping processes to the framework.

## Features

- Clean API & elegant mental model
- Concurrency supporting
- Configurable backoff retry algorithm
- Pre-request & post-response hooks supporting

## Example

```go
titleParser := func(in *goquery.Document, put remilia.Put[string]) {
    in.Find("h1").Each(func(i int, s *goquery.Selection) {
        fmt.Println(s.Text())
    })
}

rem, _ := remilia.New()
err := rem.Do(
    rem.Just("https://go.dev/"),
    rem.Unit(titleParser),
)
if err != nil {
    fmt.Println("Error: ", err)
}
```

## Install

```shell
go get -u github.com/ShroXd/remilia
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE.md) file for details.
