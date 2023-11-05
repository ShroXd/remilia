package remilia

type Option interface {
	apply(*Remilia)
}

type optionFunc func(*Remilia)

func (f optionFunc) apply(r *Remilia) {
	f(r)
}

// Name set name for scraper
func Name(name string) Option {
	return optionFunc(func(r *Remilia) {
		r.Name = name
	})
}

// ConcurrentNumber set number of goroutines for network request
func ConcurrentNumber(num int) Option {
	return optionFunc(func(r *Remilia) {
		r.ConcurrentNumber = num
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
