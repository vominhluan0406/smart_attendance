package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/config"
	"github.com/smart-attendance/smart-attendance/internal/service"
	"golang.org/x/oauth2"
)

type OAuthHandler struct {
	authService *service.AuthService
	oauthConfig *oauth2.Config
	enabled     bool
}

func NewOAuthHandler(authService *service.AuthService, cfg *config.Config) *OAuthHandler {
	enabled := cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != ""

	var oauthCfg *oauth2.Config
	if enabled {
		oauthCfg = &oauth2.Config{
			ClientID:     cfg.MicrosoftClientID,
			ClientSecret: cfg.MicrosoftClientSecret,
			RedirectURL:  cfg.MicrosoftRedirectURI,
			Scopes:       []string{"openid", "profile", "email", "User.Read"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", cfg.MicrosoftTenantID),
				TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", cfg.MicrosoftTenantID),
			},
		}
	}

	return &OAuthHandler{
		authService: authService,
		oauthConfig: oauthCfg,
		enabled:     enabled,
	}
}

// IsEnabled returns whether Microsoft OAuth is configured.
func (h *OAuthHandler) IsEnabled() bool {
	return h.enabled
}

// MicrosoftLogin redirects the user to Microsoft login page.
func (h *OAuthHandler) MicrosoftLogin(w http.ResponseWriter, r *http.Request) {
	if !h.enabled {
		http.Redirect(w, r, "/auth/login?error=Microsoft+login+is+not+configured", http.StatusFound)
		return
	}

	state := generateOAuthState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	url := h.oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// MicrosoftCallback handles the OAuth2 callback from Microsoft.
func (h *OAuthHandler) MicrosoftCallback(w http.ResponseWriter, r *http.Request) {
	if !h.enabled {
		http.Redirect(w, r, "/auth/login?error=Microsoft+login+is+not+configured", http.StatusFound)
		return
	}

	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value == "" {
		log.Printf("[handler][oauth] ERROR missing oauth state cookie")
		http.Redirect(w, r, "/auth/login?error=Invalid+OAuth+state", http.StatusFound)
		return
	}

	if r.URL.Query().Get("state") != stateCookie.Value {
		log.Printf("[handler][oauth] ERROR state mismatch")
		http.Redirect(w, r, "/auth/login?error=Invalid+OAuth+state", http.StatusFound)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Check for error from Microsoft
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		desc := r.URL.Query().Get("error_description")
		log.Printf("[handler][oauth] ERROR from Microsoft: %s — %s", errMsg, desc)
		http.Redirect(w, r, "/auth/login?error=Microsoft+login+failed", http.StatusFound)
		return
	}

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Printf("[handler][oauth] ERROR missing authorization code")
		http.Redirect(w, r, "/auth/login?error=Missing+authorization+code", http.StatusFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	token, err := h.oauthConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("[handler][oauth] ERROR exchanging code: %v", err)
		http.Redirect(w, r, "/auth/login?error=Failed+to+exchange+code", http.StatusFound)
		return
	}

	// Fetch user info from Microsoft Graph
	userInfo, err := h.fetchMicrosoftUserInfo(ctx, token)
	if err != nil {
		log.Printf("[handler][oauth] ERROR fetching user info: %v", err)
		http.Redirect(w, r, "/auth/login?error=Failed+to+get+user+info", http.StatusFound)
		return
	}

	// Login or create user
	tokens, _, err := h.authService.LoginOrCreateFromOAuth(*userInfo)
	if err != nil {
		log.Printf("[handler][oauth] ERROR login/create: %v", err)
		msg := "Login+failed"
		if err == service.ErrAccountDisabled {
			msg = "Account+is+disabled"
		}
		http.Redirect(w, r, "/auth/login?error="+msg, http.StatusFound)
		return
	}

	setTokenCookies(w, tokens)
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// microsoftUserResponse represents the Microsoft Graph /me response.
type microsoftUserResponse struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

func (h *OAuthHandler) fetchMicrosoftUserInfo(ctx context.Context, token *oauth2.Token) (*service.OAuthUserInfo, error) {
	client := h.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return nil, fmt.Errorf("request microsoft graph: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("microsoft graph returned status %d", resp.StatusCode)
	}

	var msUser microsoftUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&msUser); err != nil {
		return nil, fmt.Errorf("decode microsoft user: %w", err)
	}

	email := msUser.Mail
	if email == "" {
		email = msUser.UserPrincipalName
	}
	if email == "" {
		return nil, fmt.Errorf("no email found in Microsoft profile")
	}

	return &service.OAuthUserInfo{
		Provider: "microsoft",
		OAuthID:  msUser.ID,
		Email:    email,
		FullName: msUser.DisplayName,
	}, nil
}

func generateOAuthState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
