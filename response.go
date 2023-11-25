package remilia

import (
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

type Response struct {
	internal *http.Response
	document *goquery.Document
}
