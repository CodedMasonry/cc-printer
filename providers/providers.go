package providers

import (
	"os"
	"time"
)

type Provider interface {
	// GetAttachments fetches all files after a specific time & whether to delete
	// and returns file location
	GetAttachments(time.Time, bool) []*os.File
}

var ProviderList = []string {
	"google",
}