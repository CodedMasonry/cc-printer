package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GoogleProvider struct {
	srv *gmail.Service
}

// Google cloud project info, include at compile time
// Repo OpSec reasons
var (
	GoogleClientID     string = "COMPILE_TIME"
	GoogleClientSecret string = "COMPILE_TIME"
)

func AuthenticateUser() GoogleProvider {
	if GoogleClientID == "COMPILE_TIME" {
		panic("Google provider credentials not included; include during compile time")
	}
	
	ctx := context.Background()
	config := &oauth2.Config{
		ClientID:     GoogleClientID,
		ClientSecret: GoogleClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  "http://localhost",
		Scopes: []string{
			gmail.GmailReadonlyScope,
		},
	}

	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Gmail client: %v", err)
	}

	return GoogleProvider{
		srv: srv,
	}
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
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
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

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
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
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
	json.NewEncoder(f).Encode(token)
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
