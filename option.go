package remilia

import (
	"time"
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
		r.ConcurrentNumber = num
	})
}

// Name set name for scraper
func Name(name string) Option {
	return optionFunc(func(r *Remilia) {
		r.Name = name
	})
}

// AllowedDomains sets a string list that specifies the domains accessible to the web scraper for crawling
func AllowedDomains(domains ...string) Option {
	return optionFunc(func(r *Remilia) {
		r.AllowedDomains = domains
	})
}

// Delay sets sleep duration before web scraper sends request
func Delay(delay time.Duration) Option {
	return optionFunc(func(r *Remilia) {
		r.Delay = delay
	})
}

// UserAgent sets user agent used by request
func UserAgent(ua string) Option {
	return optionFunc(func(r *Remilia) {
		r.UserAgent = ua
	})
}

func ConsoleLog(logLevel LogLevel) Option {
	return optionFunc(func(r *Remilia) {
		r.consoleLogLevel = logLevel
	})
}

func FileLog(logLevel LogLevel) Option {
	return optionFunc(func(r *Remilia) {
		r.fileLogLevel = logLevel
	})
}
