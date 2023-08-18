package network

import "time"

type Limit struct {
	AllowedDomain string
	Delay         time.Duration
}

func NewLimit(domain string, delay time.Duration) *Limit {
	return &Limit{
		AllowedDomain: domain,
		Delay:         delay,
	}
}
