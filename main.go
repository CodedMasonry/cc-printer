package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CodedMasonry/cc-printer/providers"
	"github.com/adrg/xdg"
	"go.uber.org/zap"
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
	lastFetch     time.Time
	// The encryption key stored in byte array
	EncryptionKey []byte
}

var (
	configDir    = xdg.DataHome
	GlobalConfig *Config
	GlobalState  *State
	// When building to production set to true during compile time
	IsProduction = false
	Logger       *zap.SugaredLogger
)

func main() {
	lgr, _ := zap.NewDevelopment()
	defer lgr.Sync()
	Logger = lgr.Sugar()

	GlobalConfig = fetchConfig()
	GlobalState = fetchState()

	GlobalConfig.SaveToFile()
	defer GlobalState.SaveToFile()

	provider := providers.ProviderList[GlobalConfig.Provider]
	for {
		result := provider.GetAttachments(GlobalState.lastFetch)
		for _, file := range result {
			fmt.Printf("Attachment: %v\n", file.Name())
		}
	}
}

func fetchConfig() *Config {
	byt, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		return initConfig()
	}

	var config *Config
	err = json.Unmarshal(byt, &config)
	if err != nil {
		Logger.Panicf("Unable to read config: %v\nPlease delete config if issue persists", err)
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
		conf.AllowedSenders = append(conf.AllowedSenders, promptString("Add Allowed Sender"))
		if !promptBool("Add Another Sender?", false) {
			break
		}
	}
	conf.DeletePrinted = promptBool("Delete printed emails?", true)
	if !promptBool("Use default printer?", true) {
		conf.Printer = promptString("What printer do you want to use?")
	}
	if promptBool("Add Print flags (Using `lp` flags)", false) {
		for {
			conf.PrintFlags = append(conf.PrintFlags, promptString("Add flag"))
			if !promptBool("Add Another", true) {
				break
			}
		}
	}

	fmt.Printf("Choose an email provider")
	for _, key := range providers.ProviderList {
		fmt.Printf("\t- %s", key)
	}

	for {
		provider := strings.ToLower(promptString("Provider"))
		if _, ok := providers.ProviderList[provider]; ok {
			conf.Provider = provider
			break
		}
	}

	return conf
}

func (c *Config) SaveToFile() {
	file, err := os.OpenFile(filepath.Join(configDir, "config.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		Logger.Error("Unable to save config: ", err)
		return
	}

	data, err := json.Marshal(c)
	if err != nil {
		Logger.DPanicf("Unable to serialize config: ", err)
	}

	if _, err = file.Write(data); err != nil {
		Logger.Error("Failed to write config to file: ", err)
	}
}

func (s *State) SaveToFile() {
	file, err := os.OpenFile(filepath.Join(configDir, "state.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		Logger.Warn("Unable to save state: ", err)
		return
	}

	data, err := json.Marshal(s)
	if err != nil {
		Logger.DPanic("Unable to serialize config: ", err)
	}

	if _, err = file.Write(data); err != nil {
		Logger.Error("Failed to write state to file: ", err)
	}
}

func fetchState() *State {
	byt, err := os.ReadFile(filepath.Join(configDir, "state.json"))
	if err != nil {
		return initState()
	}

	var state *State
	err = json.Unmarshal(byt, &state)
	if err != nil {
		Logger.Warn("Unable to read state, resetting state\n")
		state = initState()
	}

	return state
}

func initState() *State {
	return &State{
		lastFetch:     time.Time{},
		EncryptionKey: genEncryptionKey(),
	}
}

func genEncryptionKey() []byte {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		Logger.Panicf("Unable to generate an encryption key: %v", err)
	}
	return bytes
}

func promptString(msg string) string {
	fmt.Printf("\n%v: ", msg)

	var str string
	if _, err := fmt.Scan(&str); err != nil {
		Logger.Error("Unable to read input: ", err)
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
	if _, err := fmt.Scan(&str); err != nil {
		Logger.Error("Unable to read input: ", err)
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
