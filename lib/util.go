package lib

import (
	"fmt"
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
	if opts.Version {
		fmt.Printf("%s: %s", Name, Version)
		os.Exit(0)
	}

	if err != nil {
		return err
	}

	if opts.TimeWindow < 0 || opts.TimeWindow > 30 {
		return fmt.Errorf("Specify a value greater than 0 days and less than 31 days for TimeWindow: ", opts.TimeWindow)
	}

	if opts.ErrorBudgetSize < 0 || opts.ErrorBudgetSize > 100 {
		return fmt.Errorf("Specify the error budget size as an integer less than 100%", opts.ErrorBudgetSize)
	}

	return nil
}
