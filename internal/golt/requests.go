package golt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"strings"
)

const (
	ClientID              = "com_jagex_auth_desktop_launcher"
	StandardLoginClientID = "1fddee4e-b100-4f4e-b2b0-097f9088f9d2"
	LoginProvider         = "runescape"
	AuthUrl               = "https://account.jagex.com/oauth2/auth"
	TokenUrl              = "https://account.jagex.com/oauth2/token"
	RedirectUrl           = "https://secure.runescape.com/m=weblogin/launcher-redirect"
	ApiUrl                = "https://api.jagex.com/v1"
	ProfileApiUrl         = "https://secure.jagex.com/rs-profile/v1"
	ShieldUrl             = "https://auth.jagex.com/shield/oauth/token"
	GameSessionApiUrl     = "https://auth.jagex.com/game-session/v1"
	OsrsBasicAuthHeader   = "Basic Y29tX2phZ2V4X2F1dGhfZGVza3RvcF9vc3JzOnB1YmxpYw=="
)

func getAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     ClientID,
		ClientSecret: "",
		Scopes: []string{
			"openid",
			"offline",
			"gamesso.token.create",
			"user.profile.read",
		},
		Endpoint: oauth2.Endpoint{
			TokenURL: TokenUrl,
			AuthURL:  AuthUrl,
		},
		RedirectURL: RedirectUrl,
	}
}

func Login() error {
	conf := getAuthConfig()

	verifier := oauth2.GenerateVerifier()
	state := makeRandomState()

	url := conf.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	err := WriteToConfigFile(state, "state")
	if err != nil {
		return err
	}
	err = WriteToConfigFile(verifier, "verifier")
	if err != nil {
		return err
	}

	return browser.OpenURL(url)
}

func Exchange(code string) (*oauth2.Token, error) {
	conf := getAuthConfig()

	verifier, err := ReadFromConfigFile("verifier")
	if err != nil {
		return nil, err
	}

	return conf.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
}

func ParseIdToken(idToken string) (map[string]interface{}, error) {
	sections := strings.Split(idToken, ".")
	if len(sections) != 3 {
		return nil, errors.New(fmt.Sprintf("malformed id_token: %d sections, expected 3", len(sections)))
	}

	// Read header
	buffer, err := base64.RawStdEncoding.DecodeString(sections[0])
	if err != nil {
		return nil, errors.New("could not parse header as base-64 string")
	}

	header, err := parseJson(buffer)
	if err != nil {
		return nil, errors.New("could not parse header as JSON")
	}

	headerType, ok := header["typ"]
	if !ok {
		return nil, errors.New("could not get header type")
	}

	if headerType != "JWT" {
		return nil, errors.New(fmt.Sprintf("bad id_token header: typ %s, expected JWT", headerType))
	}

	// Read payload
	buffer, err = base64.RawStdEncoding.DecodeString(sections[1])
	if err != nil {
		return nil, errors.New("could not parse payload as base-64 string")
	}

	payload := make(map[string]interface{})
	err = json.Unmarshal(buffer, &payload)
	if err != nil {
		return nil, errors.New("could not parse payload as base-64 string")
	}

	log.Println(payload)

	return payload, err
}

func StandardLogin(idToken string) (*Tuple[string, string], error) {
	config := getAuthConfig()

	config.ClientID = StandardLoginClientID
	config.RedirectURL = "http://localhost"
	config.Scopes = []string{"openid", "offline"}

	url := config.AuthCodeURL(
		makeRandomState(),
		oauth2.SetAuthURLParam("id_token_hint", idToken),
		// TODO: Verify nonce
		oauth2.SetAuthURLParam("nonce", uuid.New().String()),
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("response_type", "id_token code"),
	)

	// Handler to serve JavaScript code to pilfer the URL fragment
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// JavaScript code to extract URL fragment and make another request
		js := `
        <script>
            // Extract URL fragment
            var fragment = window.location.hash.substring(1);

            // Make another request with the fragment
            var xhr = new XMLHttpRequest();
            xhr.open('GET', '/process_fragment?' + fragment, true);
			xhr.onreadystatechange = function() {
                if (xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
					document.body.innerHTML = "You may now close this window";
                }
            };
            xhr.send();
        </script>
        `
		w.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprintf(w, js)
		if err != nil {
			log.Fatal(err)
		}
	})

	channel := make(chan Tuple[string, string], 1)

	// Handler to process the fragment received from JavaScript
	http.HandleFunc("/process_fragment", func(w http.ResponseWriter, r *http.Request) {
		// Get the fragment from the query parameters
		code := r.URL.Query().Get("code")
		idToken := r.URL.Query().Get("id_token")
		// Respond with a success message
		_, err := fmt.Fprintf(w, "Received")
		if err != nil {
			Die(err)
		}

		channel <- Tuple[string, string]{code, idToken}
	})

	// Start the server
	server := &http.Server{
		// TODO: sudo iptables -t nat -I OUTPUT -p tcp -d 127.0.0.1 --dport 80 -j REDIRECT --to-ports 8080
		Addr:    ":8080",
		Handler: nil,
	}
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			Die(err)
		}
	}()

	if err := browser.OpenURL(url); err != nil {
		return nil, err
	}

	// Wait for response
	tup := <-channel

	// Shut down server
	err := server.Shutdown(context.Background())
	if err != nil {
		return nil, err
	}

	return &tup, nil
}

func GetGameSession(idToken string) (string, error) {
	// Just to validate id_token
	_, err := ParseIdToken(idToken)
	if err != nil {
		return "", err
	}

	client := resty.New()

	body := map[string]interface{}{
		"idToken": idToken,
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
		// TODO: Parse this from GameSessionApiUrl
		"Host": "auth.jagex.com",
	}

	response, err := client.R().
		SetHeaders(headers).
		SetBody(body).
		Post(GameSessionApiUrl + "/sessions")

	if err != nil {
		return "", err
	}

	data := make(map[string]interface{})
	err = json.Unmarshal(response.Body(), &data)
	if err != nil {
		return "", err
	}

	sessionId, ok := data["sessionId"].(string)
	if !ok {
		return "", errors.New(fmt.Sprintf("getGameSession: did not find 'sessionId', data: %s", data))
	}

	return sessionId, nil
}

type UserDetails struct {
	DisplayName string `json:"displayName"`
	Id          string `json:"id"`
	Suffix      string `json:"suffix"`
	UserId      string `json:"userId"`
}

func GetUserDetails(sub string, accessToken string) (*UserDetails, error) {
	url := fmt.Sprintf("%s/users/%s/displayName", ApiUrl, sub)

	client := resty.New()
	client.SetHeaders(map[string]string{
		"Authorization": "Bearer " + accessToken,
	})

	response, err := client.R().Get(url)
	if err != nil {
		return nil, err
	}
	if response.StatusCode() != 200 {
		return nil, errors.New(fmt.Sprintf("could not get user details: %s", response.Body()))
	}

	info := &UserDetails{}
	err = json.Unmarshal(response.Body(), info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

type Account struct {
	AccountId   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	UserHash    string `json:"userHash"`
}

func GetAccounts(sessionId string) ([]Account, error) {
	url := fmt.Sprintf("%s/accounts", GameSessionApiUrl)

	client := resty.New()
	client.SetHeaders(map[string]string{
		"Accept":        "application/json",
		"Authorization": "Bearer " + sessionId,
	})

	response, err := client.R().Get(url)
	if err != nil {
		return nil, err
	}

	var accounts []Account
	err = json.Unmarshal(response.Body(), &accounts)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func getShieldTokens(accessToken string) error {
	client := resty.New()
	client.SetHeaders(map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": OsrsBasicAuthHeader,
	})

	params := map[string]string{
		"token":      accessToken,
		"grant_type": "token_exchange",
		"scope":      "gamesso.token.create",
	}

	url := ShieldUrl

	response, err := client.R().
		SetQueryParams(params).
		Post(url)

	if err != nil {
		return err
	}

	var data map[string]interface{}
	err = json.Unmarshal(response.Body(), &data)
	if err != nil {
		return err
	}

	log.Println("getShieldToken:", string(response.Body()))
	return nil
}
