package recover

import (
	"fmt"
	"github.com/insionng/makross"
	"github.com/insionng/makross/libraries/gommon/color"
	"github.com/insionng/makross/skipper"
	"log"
	"runtime"
)

type (
	// RecoverConfig defines the config for Recover middleware.
	RecoverConfig struct {
		// Skipper defines a function to skip middleware.
		Skipper skipper.Skipper

		// Size of the stack to be printed.
		// Optional. Default value 4KB.
		StackSize int `json:"stack_size"`

		// DisableStackAll disables formatting stack traces of all other goroutines
		// into buffer after the trace for the current goroutine.
		// Optional. Default value false.
		DisableStackAll bool `json:"disable_stack_all"`

		// DisablePrintStack disables printing stack trace.
		// Optional. Default value as false.
		DisablePrintStack bool `json:"disable_print_stack"`
	}
)

var (
	// DefaultRecoverConfig is the default Recover middleware config.
	DefaultRecoverConfig = RecoverConfig{
		Skipper:           skipper.DefaultSkipper,
		StackSize:         4 << 10, // 4 KB
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
)

// Recover returns a middleware which recovers from panics anywhere in the chain
// and handles the control to the centralized HTTPErrorHandler.
func Recover() makross.Handler {
	return RecoverWithConfig(DefaultRecoverConfig)
}

// RecoverWithConfig returns a Recover middleware with config.
// See: `Recover()`.
func RecoverWithConfig(config RecoverConfig) makross.Handler {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultRecoverConfig.Skipper
	}
	if config.StackSize == 0 {
		config.StackSize = DefaultRecoverConfig.StackSize
	}

	return func(c *makross.Context) error {
		if config.Skipper(c) {
			return c.Next()
		}

		defer func() {
			if r := recover(); r != nil {
				var err error
				switch r := r.(type) {
				case error:
					err = r
				default:
					err = fmt.Errorf("%v", r)
				}
				stack := make([]byte, config.StackSize)
				length := runtime.Stack(stack, !config.DisableStackAll)
				if !config.DisablePrintStack {
					log.Printf("[%s] %s %s\n", color.Red("PANIC RECOVER"), err, stack[:length])
				}

				c.Error(500, err.Error())

			}
		}()
		return c.Next()
	}

}
