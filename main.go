package main

import (
	"flag"
	"log"
	"log/slog"
	"time"

	"github.com/CodedMasonry/cc-printer/common"
	"github.com/CodedMasonry/cc-printer/printer"
	"github.com/CodedMasonry/cc-printer/providers"
	"github.com/CodedMasonry/cc-printer/providers/google"
)

func main() {
	reset := flag.Bool("reset", false, "Reset the Config and State of the program")
	flag.Parse()

	common.InitLogging()

	if *reset {
		common.DeleteAppState()
	}
	common.GlobalConfig = common.FetchConfig()
	common.GlobalState = common.FetchState()

	common.GlobalConfig.SaveToFile()
	common.GlobalState.SaveToFile()

	slog.Info("State successfully initialized", "ConfigDir", common.ConfigDir)

	provider := fetchProvider(common.GlobalConfig.Provider)
	for {
		slog.Debug("Fetching files", "last", common.GlobalState.LastFetch)
		result := provider.GetAttachments(common.GlobalState.LastFetch, common.GlobalConfig.DeletePrinted)
		slog.Debug("Finished fetching", "files", len(result))

		for _, file := range result {
			slog.Info("Attachment Downloaded", "file", file.Name())
			printer.PrintFile(file)
		}

		common.GlobalState.LastFetch = time.Now().UTC()
		go common.GlobalState.SaveToFile()
		time.Sleep(1 * time.Minute)
	}
}

func fetchProvider(provider string) providers.Provider {
	switch provider {
	case "google":
		return google.InitProvider(common.GlobalConfig.DeletePrinted, common.GlobalConfig.AllowedSenders)
	default:
		log.Fatal("Unknown or Unsupported provider: ", provider)
	}

	panic("Unreachable")
}
