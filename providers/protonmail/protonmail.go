package protonmail

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ProtonMail/go-proton-api"
	"github.com/gofiber/fiber/v2/log"
	"golang.org/x/term"
)

type ProtonProvider struct {
	client         *proton.Client
	UID            string
	refresh        string
	allowedSenders []string
}

func InitProvider(allowedSenders []string) *ProtonProvider {
	client, UID, refresh := authenticate()
	return &ProtonProvider{
		client,
		UID,
		refresh,
		allowedSenders,
	}
}

func authenticate() (*proton.Client, string, string) {
	p := proton.New()
	ctx := context.Background()

	c, auth, err := p.NewClientWithLogin(ctx, getInput("Username"), getPasswd("Password"))
	if err != nil {
		log.Fatal("Failed to authenticate: ", err)
	}
	defer c.Close()

	if auth.TwoFA.Enabled&proton.HasTOTP != 0 {
		if err := c.Auth2FA(ctx, proton.Auth2FAReq{TwoFactorCode: getInput("2FA Code")}); err != nil {
			panic(err)
		}
	}
	return c, auth.UID, auth.RefreshToken
}

func getInput(prompt string) string {
	fmt.Printf("\n%v: ", prompt)
	var input string
	fmt.Scanln(&input)
	return input
}

func getPasswd(prompt string) []byte {
	fmt.Printf("\n%v: ", prompt)
	input, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatal("Failed to read password: ", err)
	}

	return input
}

func (p *ProtonProvider) GetAttachments(after time.Time, deleteFetced bool) []*os.File {
	ctx := context.Background()
	messages, err := p.client.GetMessageMetadata(ctx, proton.MessageFilter{})
	if err != nil {
		log.Fatal("Failed to fetch messages: ", err)
	}
	for _, msg  := range messages {
		if msg.Time
	}
	return []*os.File{}
}
