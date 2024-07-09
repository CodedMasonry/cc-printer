package main

import (
	"log"
	"log/slog"

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
	result := provider.GetAttachments(common.GlobalState.LastFetch)
	for _, file := range result {
		slog.Info("Attachment Downloaded", "file", file.Name())
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