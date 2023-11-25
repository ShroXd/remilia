package remilia

import "github.com/PuerkitoBio/goquery"

type (
	URLParser    func(*goquery.Document)
	HTMLParser   func(*goquery.Document)
	DataConsumer func(data <-chan interface{})
)

type Step struct {
	urlGenerator  URLParser
	htmlProcessor HTMLParser
	dataConsumer  DataConsumer
	// TODO: add options to control the request
}

func NewStep(urlGenerator URLParser, htmlProcessor HTMLParser, dataConsumer DataConsumer) *Step {
	return &Step{
		urlGenerator:  urlGenerator,
		htmlProcessor: htmlProcessor,
		dataConsumer:  dataConsumer,
	}
}
