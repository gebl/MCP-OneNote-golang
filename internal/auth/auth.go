package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// OAuth2Config holds Microsoft Identity Platform OAuth2 configuration for public clients (PKCE)
// TODO: Implement full OAuth2 flow, token storage, and refresh logic

type OAuth2Config struct {
	ClientID    string
	TenantID    string
	RedirectURI string
}

// TokenManager will handle access/refresh tokens
// TODO: Implement secure token storage and refresh

type TokenManager struct {
	AccessToken  string
	RefreshToken string
	Expiry       int64 // Unix timestamp
}

// NewOAuth2Config creates a new OAuth2Config from config
func NewOAuth2Config(clientID, tenantID, redirectURI string) *OAuth2Config {
	log.Println("[auth] Creating OAuth2Config (PKCE public client)...")
	return &OAuth2Config{
		ClientID:    clientID,
		TenantID:    tenantID,
		RedirectURI: redirectURI,
	}
}

// PKCE helpers
func GenerateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func CodeChallengeS256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// GetAuthURL generates the Microsoft login URL for user consent (PKCE)
func (c *OAuth2Config) GetAuthURL(state, codeChallenge string) string {
	authURL := fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?client_id=%s&response_type=code&redirect_uri=%s&response_mode=query&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		url.PathEscape(c.TenantIDOrCommon()),
		url.QueryEscape(c.ClientID),
		url.QueryEscape(c.RedirectURI),
		url.QueryEscape("offline_access Notes.ReadWrite"),
		url.QueryEscape(state),
		url.QueryEscape(codeChallenge),
	)
	log.Printf("[auth] Auth URL: %s", authURL)
	return authURL
}

// TenantIDOrCommon returns the tenant ID or "common" if not set
func (c *OAuth2Config) TenantIDOrCommon() string {
	if c.TenantID == "" {
		return "common"
	}
	return c.TenantID
}

// ExchangeCode exchanges the auth code for access and refresh tokens (PKCE)
func (c *OAuth2Config) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenManager, error) {
	log.Println("[auth] Exchanging authorization code for tokens...")
	endpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.TenantIDOrCommon())
	data := url.Values{}
	data.Set("client_id", c.ClientID)
	data.Set("scope", "offline_access Notes.ReadWrite")
	data.Set("code", code)
	data.Set("redirect_uri", c.RedirectURI)
	data.Set("grant_type", "authorization_code")
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		log.Printf("[auth] Error creating token request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[auth] Error sending token request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[auth] Token exchange failed: %s", string(body))
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}
	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Printf("[auth] Error decoding token response: %v", err)
		return nil, err
	}
	log.Println("[auth] Token exchange successful.")
	return &TokenManager{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		Expiry:       time.Now().Unix() + res.ExpiresIn,
	}, nil
}

// StartLocalServer starts a local HTTP server to capture the auth code
func StartLocalServer(redirectPath string, codeCh chan<- string, state string) (*http.Server, error) {
	log.Printf("[auth] Starting local HTTP server at http://localhost:8080%s to receive auth code...", redirectPath)
	mux := http.NewServeMux()
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			log.Println("[auth] Invalid state received in redirect.")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			log.Println("[auth] Missing code in redirect.")
			return
		}
		w.Write([]byte("Authentication successful. You may close this window."))
		log.Println("[auth] Received authorization code from redirect.")
		codeCh <- code
	})
	server := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		_ = server.ListenAndServe()
	}()
	return server, nil
}

// SaveTokens saves tokens to a file (for demo; use secure storage in production)
func (tm *TokenManager) SaveTokens(path string) error {
	log.Printf("[auth] Saving tokens to %s...", path)
	f, err := os.Create(path)
	if err != nil {
		log.Printf("[auth] Error saving tokens: %v", err)
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tm)
}

// LoadTokens loads tokens from a file
func LoadTokens(path string) (*TokenManager, error) {
	log.Printf("[auth] Loading tokens from %s...", path)
	f, err := os.Open(path)
	if err != nil {
		log.Printf("[auth] No token file found at %s.", path)
		return nil, err
	}
	defer f.Close()
	tm := &TokenManager{}
	if err := json.NewDecoder(f).Decode(tm); err != nil {
		log.Printf("[auth] Error decoding token file: %v", err)
		return nil, err
	}
	log.Println("[auth] Tokens loaded from file.")
	return tm, nil
}

// RefreshToken refreshes the access token using the refresh token
func (c *OAuth2Config) RefreshToken(ctx context.Context, refreshToken string) (*TokenManager, error) {
	log.Println("[auth] Refreshing access token using refresh token...")
	endpoint := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.TenantIDOrCommon())
	data := url.Values{}
	data.Set("client_id", c.ClientID)
	data.Set("scope", "offline_access Notes.ReadWrite")
	data.Set("refresh_token", refreshToken)
	data.Set("redirect_uri", c.RedirectURI)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		log.Printf("[auth] Error creating refresh request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[auth] Error sending refresh request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[auth] Token refresh failed: %s", string(body))
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}
	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Printf("[auth] Error decoding refresh response: %v", err)
		return nil, err
	}
	log.Println("[auth] Token refresh successful.")
	return &TokenManager{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		Expiry:       time.Now().Unix() + res.ExpiresIn,
	}, nil
}

// IsExpired returns true if the access token is expired
func (tm *TokenManager) IsExpired() bool {
	return time.Now().Unix() > tm.Expiry-60 // 60s buffer
}
