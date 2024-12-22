package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chowder/golt/internal/golt"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"io"
	"log"
	"os"
	"time"
)

func launchGameWithToken(token *oauth2.Token, idToken string) error {
	idTokenPayload, err := golt.ParseIdToken(idToken)
	if err != nil {
		return err
	}

	// Get user details
	sub := idTokenPayload["sub"].(string)
	user, err := golt.GetUserDetails(sub, token.AccessToken)
	if err != nil {
		return err
	}
	log.Println(fmt.Sprintf("You are logged in as: %s#%s", user.DisplayName, user.Suffix))

	loginProvider, ok := idTokenPayload["login_provider"]
	if !ok || loginProvider != golt.LoginProvider {
		log.Println("No known login_provider found in JWT token, using standard login...")

		tup, err := golt.StandardLogin(idToken)
		if err != nil {
			return err
		}

		err = golt.WriteToConfigFile(tup.Second, "id_token")
		if err != nil {
			return err
		}

		sessionId, err := golt.GetGameSession(tup.Second)
		if err != nil {
			return err
		}

		accounts, err := golt.GetAccounts(sessionId)
		if err != nil {
			return err
		}
		log.Println("Found", len(accounts), "accounts")

		account, err := golt.GetChosenAccount(user, accounts)
		if err != nil {
			if errors.Is(err, &golt.NoAccountChosenError{}) {
				log.Println("No account was chosen")
				return nil
			}
			return err
		}

		err = golt.LaunchGame(sessionId, account)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleSocialAuth(payload map[string]string) error {
	token, err := golt.Exchange(payload["code"])
	if err != nil {
		return err
	}

	idToken := token.Extra("id_token").(string)
	err = golt.WriteToConfigFile(idToken, "id_token")
	if err != nil {
		return err
	}

	// OAuth token
	tokenAsJson, err := json.Marshal(token)
	if err != nil {
		return err
	}

	err = golt.WriteToConfigFile(string(tokenAsJson), "token")
	if err != nil {
		return err
	}

	return launchGameWithToken(token, idToken)
}

func getCachedTokens() (*golt.Tuple[*oauth2.Token, string], error) {
	tokenJson, err := golt.ReadFromConfigFile("token")
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	err = json.Unmarshal([]byte(tokenJson), &token)
	if err != nil {
		return nil, err
	}

	// TODO: try to refresh token (see: https://github.com/golang/oauth2/issues/84#issuecomment-254576796)
	currentTime := time.Now()
	if currentTime.Sub(token.Expiry) <= 30*time.Minute {
		return nil, errors.New("cached token has expired")
	}

	idToken, err := golt.ReadFromConfigFile("id_token")
	if err != nil {
		return nil, err
	}

	return &golt.Tuple[*oauth2.Token, string]{
		First:  &token,
		Second: idToken,
	}, nil
}

func handleRegularLaunch() error {
	tup, err := getCachedTokens()
	if err != nil {
		log.Println("Could not load cached tokens, initiating regular login flow")
		err := golt.Login()
		return err
	}

	log.Println("Loaded cached tokens")
	err = launchGameWithToken(tup.First, tup.Second)
	if err != nil {
		log.Println("Could not login with cached tokens, initiating regular login flow")
		err := golt.Login()
		return err
	}

	return nil
}

func main() {
	// Ignore stdout/stderr from browser - otherwise this messes up the bubbletea TUI
	browser.Stdout = io.Discard
	browser.Stderr = io.Discard

	if len(os.Args) < 2 {
		err := handleRegularLaunch()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	payload := golt.ParseIntentPayload(os.Args[1])

	intent, ok := payload["intent"]
	if !ok {
		log.Fatal("No intent found in payload", payload)
	}

	if intent == "social_auth" {
		err := handleSocialAuth(payload)
		if err != nil {
			golt.Die(err)
		}
	}

	log.Println("Press Enter to exit...")
	fmt.Scanln()
}
