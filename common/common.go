package common

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

var (
	// When building to production set to true during compile time
	IsProduction = false
	CallbackPort = ":8080"
)

func InitLogging() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.RFC822,
			NoColor: !isatty.IsTerminal(os.Stderr.Fd()),
		}),
	))
}
