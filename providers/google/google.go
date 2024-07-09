package google

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/CodedMasonry/cc-printer/common"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GoogleProvider struct {
	srv            *gmail.Service
	deletePrinted  bool
	allowedSenders []string
}

// Google cloud project info, include at compile time
// Repo OpSec reasons
var (
	GoogleClientID     string
	GoogleClientSecret string
	GoogleCallbackURL  = "/auth/callback/google"
)

func InitProvider(deletePrinted bool, allowedSenders []string) *GoogleProvider {
	return &GoogleProvider{
		srv:            AuthenticateUser(),
		deletePrinted:  deletePrinted,
		allowedSenders: allowedSenders,
	}
}

func AuthenticateUser() *gmail.Service {
	if GoogleClientID == "" || GoogleClientSecret == "" {
		panic("Google provider credentials not included; include during compile time")
	}

	ctx := context.Background()
	config := &oauth2.Config{
		ClientID:     GoogleClientID,
		ClientSecret: GoogleClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost:8080" + GoogleCallbackURL,
		Scopes: []string{
			gmail.GmailReadonlyScope,
		},
	}

	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	return srv
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := filepath.Join(common.ConfigDir, "token.json")
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\nOpen the following link in your browser to authenticate: \n%v", authURL)
	quit := make(chan bool)
	result := make(chan string)
	go authInput(quit, result)
	go authCallback(quit, result)

	authCode := <-result
	close(quit)

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		log.Fatal("unable to read token file", err)
	}

	encrypted := make([]byte, info.Size())
	_, err = f.Read(encrypted)
	if err != nil {
		log.Fatal("Unable to read", err)
	}

	decrypted := decryptBytes(encrypted)
	tok := &oauth2.Token{}
	err = json.NewDecoder(bytes.NewReader(decrypted)).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()

	bytes, err := json.Marshal(token)
	if err != nil {
		log.Fatalf("Unable to encode token: %v", err)
	}

	encrypted := encryptBytes(bytes)
	f.Write(encrypted)
}

func authInput(quit chan bool, result chan string) {
	fmt.Print("\nAuthorization code: ")
	var input string
	fmt.Scanln(&input)
	select {
	case <-quit:
		return
	case result <- input:
		return
	}
}

func authCallback(quit chan bool, result chan string) {
	http.HandleFunc("/auth/callback/google", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprintf(w, "Invalid Authentication Code\n")
			log.Fatal("Invalid authentication code")
		}
		fmt.Fprintf(w, "Successfully Authenticated, You may close this tab\n")
		result <- code
	})

	server := &http.Server{Addr: common.CallbackPort, Handler: http.DefaultServeMux}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Authentication callback failed", "error", err)
		}
	}()

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Failed to shutdown authentication callback", "error", err)
	}
}

func encryptBytes(msg []byte) []byte {
	aead, err := chacha20poly1305.NewX(common.GlobalState.EncryptionKey)
	if err != nil {
		log.Panic(err)
	}

	nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(msg)+aead.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		log.Panic(err)
	}

	return aead.Seal(nonce, nonce, msg, nil)
}

func decryptBytes(encrypted []byte) []byte {
	aead, err := chacha20poly1305.NewX(common.GlobalState.EncryptionKey)
	if err != nil {
		log.Panic(err)
	}

	if len(encrypted) < aead.NonceSize() {
		log.Panic("Ciphertext too short")
	}

	nonce, ciphertext := encrypted[:aead.NonceSize()], encrypted[aead.NonceSize():]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Panic(err)
	}
	return plaintext
}

func (p GoogleProvider) GetAttachments(after time.Time) []*os.File {
	user := "me"
	r, err := p.srv.Users.Messages.List(user).LabelIds("INBOX").Q("from:(sundoesdevelopment@gmail.com)").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve labels: %v", err)
	}

	files := make([]*os.File, 0)
	for _, msg := range r.Messages {
		message, err := p.srv.Users.Messages.Get(user, msg.Id).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve message: %v: %v\n", msg.Id, err)
		}
		files = slices.Concat(files, p.parseAttachments(message))
	}
	return files
}

func (p GoogleProvider) parseAttachments(message *gmail.Message) []*os.File {
	files := make([]*os.File, 0)
	for _, part := range message.Payload.Parts {
		if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
			attach, err := p.srv.Users.Messages.Attachments.Get("me", message.Id, part.Body.AttachmentId).Do()
			if err != nil {
				log.Printf("Unable to retrieve attachment: %v", err)
				continue
			}

			data, err := base64.URLEncoding.DecodeString(attach.Data)
			if err != nil {
				log.Printf("Unable to decode attachment data: %v", err)
				continue
			}

			f, err := os.CreateTemp("", "*."+nameToType(part.Filename))
			if err != nil {
				log.Fatalf("Unable to save temporary file: %v", err)
			}

			_, err = f.Write(data)
			if err != nil {
				log.Printf("Unable to write to temporary file: %v", err)
			}

			files = append(files, f)
		}
	}

	return files
}

// Parses a file name (Ex: something.png) to a file type (Ex: png)
func nameToType(str string) string {
	split := strings.Split(str, ".")
	return split[len(split)-1]
}
