package debug

import (
	"utils/logging"
)

var Logger *logging.Writer

func SetLogger(logger *logging.Writer) {
	Logger = logger
}
