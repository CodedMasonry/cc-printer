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
	"github.com/CodedMasonry/cc-printer/providers/protonmail"
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
	refreshTime := time.Now().Add(12 * time.Hour)
	for {
		slog.Debug("Fetching files", "last", common.GlobalState.LastFetch)
		result := provider.GetAttachments(common.GlobalState.LastFetch, common.GlobalConfig.DeletePrinted)

		for _, file := range result {
			slog.Info("Attachment Downloaded", "file", file.Name())
			printer.PrintFile(file)
		}

		common.GlobalState.LastFetch = time.Now().UTC()
		go common.GlobalState.SaveToFile()

		// Restart the provider service if it has been live for over 12 hours (to force a token refresh)
		if time.Now().After(refreshTime) {
			provider = fetchProvider(common.GlobalConfig.Provider)
		}

		time.Sleep(1 * time.Minute)
	}
}

func fetchProvider(provider string) providers.Provider {
	switch provider {
	case "google":
		return google.InitProvider(common.GlobalConfig.DeletePrinted, common.GlobalConfig.AllowedSenders)
	case "protonmail":
		return protonmail.InitProvider(common.GlobalConfig.AllowedSenders)
	default:
		log.Fatal("Unknown or Unsupported provider: ", provider)
	}
	panic("Unreachable")
}
