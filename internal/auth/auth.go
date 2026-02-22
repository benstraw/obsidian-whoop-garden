package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"
)

const (
	tokenURL    = "https://api.prod.whoop.com/oauth/oauth2/token"
	authURL     = "https://api.prod.whoop.com/oauth/oauth2/auth"
	tokenFile   = "tokens.json"
	callbackPort = ":3000"
)

// TokenResponse holds OAuth token data returned by WHOOP.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// randomState generates a cryptographically random hex state string.
func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// StartAuthFlow runs the full OAuth 2.0 authorization code flow.
// It opens the browser, starts a local callback server, exchanges the code
// for tokens, and saves them to disk.
func StartAuthFlow() error {
	clientID := os.Getenv("WHOOP_CLIENT_ID")
	redirectURI := os.Getenv("WHOOP_REDIRECT_URI")

	if clientID == "" || os.Getenv("WHOOP_CLIENT_SECRET") == "" {
		return fmt.Errorf(`WHOOP API credentials are not configured.

Create a .env file in the same directory as the binary with:

  WHOOP_CLIENT_ID=your_client_id
  WHOOP_CLIENT_SECRET=your_client_secret
  WHOOP_REDIRECT_URI=http://localhost:3000/callback

You can obtain free credentials by creating an app at:
  https://developer.whoop.com/`)
	}

	state, err := randomState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	scopes := "offline read:profile read:body_measurement read:cycles read:recovery read:sleep read:workout"

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scopes)
	params.Set("state", state)

	fullAuthURL := authURL + "?" + params.Encode()

	fmt.Println("Opening browser for WHOOP authorization...")
	fmt.Println("If the browser does not open, visit:", fullAuthURL)

	_ = exec.Command("open", fullAuthURL).Start()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Addr: callbackPort, Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("state"); got != state {
			errCh <- fmt.Errorf("state mismatch (got %q)", got)
			fmt.Fprintln(w, "Authorization failed (state mismatch). You may close this tab.")
			return
		}
		code := q.Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback: %s", r.URL.RawQuery)
			fmt.Fprintln(w, "Authorization failed. You may close this tab.")
			return
		}
		fmt.Fprintln(w, "Authorization successful! You may close this tab.")
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	fmt.Printf("Waiting for OAuth callback on http://localhost%s/callback ...\n", callbackPort)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("auth flow timed out after 5 minutes")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)

	tokens, err := exchangeCode(code, redirectURI)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}

	if err := SaveTokens(tokens); err != nil {
		return fmt.Errorf("failed to save tokens: %w", err)
	}

	fmt.Printf("Authenticated successfully. Tokens saved to %s\n", tokenFile)
	return nil
}

// exchangeCode trades an authorization code for tokens.
func exchangeCode(code, redirectURI string) (TokenResponse, error) {
	clientID := os.Getenv("WHOOP_CLIENT_ID")
	clientSecret := os.Getenv("WHOOP_CLIENT_SECRET")

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	return postTokenRequest(data)
}

// postTokenRequest sends a POST to the token endpoint and decodes the response.
func postTokenRequest(data url.Values) (TokenResponse, error) {
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return TokenResponse{}, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}

	var tokens TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return TokenResponse{}, fmt.Errorf("failed to decode token response: %w", err)
	}

	tokens.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	return tokens, nil
}

// SaveTokens writes tokens to tokens.json.
func SaveTokens(tokens TokenResponse) error {
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tokenFile, data, 0600)
}

// LoadTokens reads tokens from tokens.json.
func LoadTokens() (TokenResponse, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("tokens not found (run 'auth' first): %w", err)
	}
	var tokens TokenResponse
	if err := json.Unmarshal(data, &tokens); err != nil {
		return TokenResponse{}, fmt.Errorf("failed to parse tokens.json: %w", err)
	}
	return tokens, nil
}

// RefreshIfNeeded checks token expiry and refreshes if necessary.
// Returns the valid access token.
func RefreshIfNeeded() (string, error) {
	tokens, err := LoadTokens()
	if err != nil {
		return "", err
	}

	// Refresh if expiring within 5 minutes.
	if time.Now().Add(5 * time.Minute).Before(tokens.ExpiresAt) {
		return tokens.AccessToken, nil
	}

	fmt.Println("Access token expiring soon, refreshing...")
	refreshed, err := refreshTokens(tokens.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("token refresh failed: %w", err)
	}

	if err := SaveTokens(refreshed); err != nil {
		return "", fmt.Errorf("failed to save refreshed tokens: %w", err)
	}

	return refreshed.AccessToken, nil
}

// refreshTokens exchanges a refresh token for a new token set.
func refreshTokens(refreshToken string) (TokenResponse, error) {
	clientID := os.Getenv("WHOOP_CLIENT_ID")
	clientSecret := os.Getenv("WHOOP_CLIENT_SECRET")

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	return postTokenRequest(data)
}
