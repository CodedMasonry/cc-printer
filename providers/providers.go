package providers

import (
	"os"
	"time"

	"github.com/CodedMasonry/cc-printer/providers/google"
)

type Provider interface {
	// GetAttachments fetches all files after a specific time
	// and returns file location
	GetAttachments(time.Time) []*os.File
}

var ProviderList = map[string]Provider {
	"google": google.AuthenticateUser(),
}