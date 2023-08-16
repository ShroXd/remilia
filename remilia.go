package remilia

import "remilia/pkg/logger"

type Remilia struct{}

func New() *Remilia {
	// Init the logger
	logger.New()

	return &Remilia{}
}
