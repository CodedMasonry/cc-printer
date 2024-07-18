package main

import (
	"log"
	"log/slog"
	"time"

	"github.com/CodedMasonry/cc-printer/common"
	"github.com/CodedMasonry/cc-printer/providers"
	"github.com/CodedMasonry/cc-printer/providers/google"
)

func main() {
	common.InitLogging()
	common.GlobalConfig = common.FetchConfig()
	common.GlobalState = common.FetchState()

	common.GlobalConfig.SaveToFile()
	defer common.GlobalState.SaveToFile()

	slog.Info("State successfully initialized", "ConfigDir", common.ConfigDir)

	provider := fetchProvider(common.GlobalConfig.Provider)
	for {
		slog.Debug("Fetching provider for files")
		result := provider.GetAttachments(common.GlobalState.LastFetch, common.GlobalConfig.DeletePrinted)
		slog.Debug("Finished fetching", "number of files", len(result))

		for _, file := range result {
			slog.Info("Attachment Downloaded", "file", file.Name())
		}

		common.GlobalState.LastFetch = time.Now().UTC()
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
