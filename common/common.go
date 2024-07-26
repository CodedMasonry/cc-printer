package common

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/CodedMasonry/cc-printer/providers"
	"github.com/adrg/xdg"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

var (
	// When building to production set to true during compile time
	IsProduction = false
	CallbackPort = ":8080"
	ConfigDir    = filepath.Join(xdg.DataHome, "cc-printer")

	GlobalConfig *Config
	GlobalState  *State
)

// Config for the printer
type Config struct {
	// AllowedSenders specifies only to prints files from Allowed Senders
	AllowedSenders []string
	// DeletePrinted specifies whether the email should be deleted after processing
	DeletePrinted bool
	// Printer is the printer to use; set to default to use default
	Printer string
	// PrintFlags are the flags to pass to CUPS (lp) command during printing
	PrintFlags []string
	// Provider used to pull emails from
	Provider string
	// Whether to reset state & config on launch
	Reset bool
}

type State struct {
	// lastFetch Last time the code fetched
	LastFetch time.Time
	// The encryption key stored in byte array
	EncryptionKey []byte
}

func FetchConfig() *Config {
	byt, err := os.ReadFile(filepath.Join(ConfigDir, "config.json"))
	if err != nil {
		return initConfig()
	}

	var config *Config
	err = json.Unmarshal(byt, &config)
	if err != nil {
		log.Fatalf("Unable to read config: %v\nPlease delete config if issue persists", err)
	}

	if config.Reset {
		return initConfig()
	}
	return config
}

func initConfig() *Config {
	conf := &Config{
		AllowedSenders: make([]string, 0),
		DeletePrinted:  true,
		Printer:        "default",
		PrintFlags:     make([]string, 0),
		Provider:       "google", // Defaults to google
		Reset:          false,
	}
	for {
		conf.AllowedSenders = append(conf.AllowedSenders, promptString("Allowed Sender"))
		if !promptBool("Add Another?", false) {
			break
		}
	}
	conf.DeletePrinted = promptBool("Delete printed emails?", true)
	if !promptBool("Use default printer?", true) {
		conf.Printer = promptString("What printer do you want to use?")
	}
	if promptBool("Add Print flags (Using `lp` flags)?", false) {
		for {
			conf.PrintFlags = append(conf.PrintFlags, promptString("Add flag"))
			if !promptBool("Add Another", true) {
				break
			}
		}
	}

	for {
		fmt.Printf("\nChoose an email provider")
		for _, key := range providers.ProviderList {
			fmt.Printf("\n\t- %s", key)
			if key == "google" {
				fmt.Print(" (default)")
			}
		}

		provider := strings.ToLower(promptString("Provider"))
		if provider == "" {
			conf.Provider = "google"
		}
		if slices.Contains(providers.ProviderList, provider) {
			conf.Provider = provider
			break
		}
	}

	return conf
}

func FetchState() *State {
	byt, err := os.ReadFile(filepath.Join(ConfigDir, "state.json"))
	if err != nil {
		return initState()
	}

	var state *State
	err = json.Unmarshal(byt, &state)
	if err != nil {
		slog.Warn("Unable to read state, resetting state")
		state = initState()
	}

	return state
}

func initState() *State {
	return &State{
		LastFetch:     time.Unix(1,0),
		EncryptionKey: genEncryptionKey(),
	}
}

func DeleteAppState() {
	if err := os.Remove(filepath.Join(ConfigDir, "config.json")); err != nil {
		log.Fatalf("Failed to delete config: %v", err)
	}

	if err := os.Remove(filepath.Join(ConfigDir, "state.json")); err != nil {
		log.Fatalf("Failed to delete config: %v", err)
	}
}

func genEncryptionKey() []byte {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Unable to generate an encryption key: ", err)
	}
	return bytes
}

func InitLogging() {
	var level slog.Level
	if IsProduction {
		level = slog.LevelInfo
	} else {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			TimeFormat: time.RFC822,
			NoColor:    !isatty.IsTerminal(os.Stderr.Fd()),
		}),
	))
}

func promptString(msg string) string {
	fmt.Printf("\n%v: ", msg)

	var str string
	if _, err := fmt.Scanln(&str); err != nil && err.Error() != "unexpected newline" {
		log.Fatal("Unable to read user input: ", err)
	}
	return strings.TrimSpace(str)
}

func promptBool(msg string, defaultTrue bool) bool {
	if defaultTrue {
		fmt.Printf("\n%v [Y/n]:", msg)
	} else {
		fmt.Printf("\n%v [y/N]:", msg)
	}

	var str string
	if _, err := fmt.Scanln(&str); err != nil && err.Error() != "unexpected newline" {
		log.Fatal("Unable to read user input: ", err)
	}

	str = strings.ToLower(str)
	if strings.Contains(str, "y") {
		return true
	} else if strings.Contains(str, "n") {
		return false
	} else {
		return defaultTrue
	}
}

func (c *Config) SaveToFile() {
	os.MkdirAll(ConfigDir, 0755)
	file, err := os.OpenFile(filepath.Join(ConfigDir, "config.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		slog.Error("Unable to save config", "error", err)
		return
	}

	data, err := json.Marshal(c)
	if err != nil {
		log.Fatal("Unable to serialize config: ", "error", err)
	}

	if _, err = file.Write(data); err != nil {
		slog.Error("Failed to write config to file", "error", err)
	}
}

func (s *State) SaveToFile() {
	file, err := os.OpenFile(filepath.Join(ConfigDir, "state.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		slog.Warn("Unable to save state", "error", err)
		return
	}

	data, err := json.Marshal(s)
	if err != nil {
		log.Fatal("Unable to serialize config: ", err)
	}

	if _, err = file.Write(data); err != nil {
		slog.Error("Failed to write state to file", "error", err)
	}
}
