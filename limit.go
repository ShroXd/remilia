package remilia

import "time"

type limit struct {
	AllowedDomain string
	Delay         time.Duration
}

func newLimit(domain string, delay time.Duration) *limit {
	return &limit{
		AllowedDomain: domain,
		Delay:         delay,
	}
}
