package lib

import (
	"os"

	"github.com/jessevdk/go-flags"
	"go.uber.org/zap/zapcore"
)

func calcTimeWindowMinutes(m int) float64 {
	return float64(m * 24 * 60)
}

func getMackerelToke() string {
	return os.Getenv("MACKEREL_TOKEN")
}

func initLogger() {
	if opts.Debug {
		logger = NewLogger(zapcore.DebugLevel)
	} else {
		logger = NewLogger(zapcore.InfoLevel)
	}
}

func parseArgs(args []string) error {
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return err
	}
	return nil
}
