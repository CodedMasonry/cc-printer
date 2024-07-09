package providers

import (
	"os"
	"time"
)

type Provider interface {
	// GetAttachments fetches all files after a specific time
	// and returns file location
	GetAttachments(time.Time) []*os.File
}

var ProviderList = []string {
	"google",
}