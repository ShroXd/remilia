package remilia

import "github.com/PuerkitoBio/goquery"

type (
	URLParser    func(*goquery.Document) string
	HTMLParser   func(*goquery.Document) interface{}
	DataConsumer func(data <-chan interface{})
)

type Stage struct {
	urlGenerator  URLParser
	htmlProcessor HTMLParser
	dataConsumer  DataConsumer
	// TODO: add options to control the request
}

func NewStage(urlGenerator URLParser, htmlProcessor HTMLParser, dataConsumer DataConsumer) *Stage {
	return &Stage{
		urlGenerator:  urlGenerator,
		htmlProcessor: htmlProcessor,
		dataConsumer:  dataConsumer,
	}
}
