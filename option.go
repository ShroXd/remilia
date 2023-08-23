package remilia

import (
	"time"

	"go.uber.org/zap"
)

type Option interface {
	apply(*Remilia)
}

type optionFunc func(*Remilia)

func (f optionFunc) apply(r *Remilia) {
	f(r)
}

// ConcurrentNumber set number of goroutines for network request
func ConcurrentNumber(num int) Option {
	return optionFunc(func(r *Remilia) {
		r.logger.Debug("Apply configuration: concurrent number", zap.Int("value", num))
		r.ConcurrentNumber = num
	})
}

// Name set name for scraper
func Name(name string) Option {
	return optionFunc(func(r *Remilia) {
		r.logger.Debug("Apply configuration: name", zap.String("value", name))
		r.Name = name
	})
}

// AllowedDomains sets a string list that specifies the domains accessible to the web scraper for crawling
func AllowedDomains(domains ...string) Option {
	return optionFunc(func(r *Remilia) {
		r.logger.Debug("Apply configuration: domains", zap.Any("value", domains))
		r.AllowedDomains = domains
	})
}

// Delay sets sleep duration before web scraper sends request
func Delay(delay time.Duration) Option {
	return optionFunc(func(r *Remilia) {
		r.logger.Debug("Apply configuration: delay", zap.Duration("value", delay))
		r.Delay = delay
	})
}

// UserAgent sets user agent used by request
func UserAgent(ua string) Option {
	return optionFunc(func(r *Remilia) {
		r.logger.Debug("Apply configuration: user agent", zap.String("value", ua))
		r.UserAgent = ua
	})
}
