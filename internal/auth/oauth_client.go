package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

// httpOAuthClient implements OAuthClient over golang.org/x/oauth2 with an
// explicit provider authorization/token endpoint pair plus a userinfo
// fetch, isolating the only place this package touches the network or the
// oauth2 SDK. PKCE (S256) is always used; provider client secrets never
// leave this file.
type httpOAuthClient struct {
	config      oauth2.Config
	userInfoURL string
	extractInfo func(body []byte) (OAuthUserInfo, error)
	httpClient  *http.Client
}

// NewGoogleOAuthClient returns an OAuthClient for Google Sign-In using the
// OpenID userinfo endpoint to resolve the stable subject/email.
func NewGoogleOAuthClient(clientID, clientSecret, redirectURL string) OAuthClient {
	return &httpOAuthClient{
		config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     endpoints.Google,
			Scopes:       []string{"openid", "email"},
		},
		userInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
		extractInfo: func(body []byte) (OAuthUserInfo, error) {
			var payload struct {
				Sub   string `json:"sub"`
				Email string `json:"email"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				return OAuthUserInfo{}, err
			}
			return OAuthUserInfo{Subject: payload.Sub, Email: payload.Email}, nil
		},
		httpClient: http.DefaultClient,
	}
}

// NewGitHubOAuthClient returns an OAuthClient for GitHub using the REST
// /user endpoint to resolve the stable numeric subject/email.
func NewGitHubOAuthClient(clientID, clientSecret, redirectURL string) OAuthClient {
	return &httpOAuthClient{
		config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     endpoints.GitHub,
			Scopes:       []string{"read:user", "user:email"},
		},
		userInfoURL: "https://api.github.com/user",
		extractInfo: func(body []byte) (OAuthUserInfo, error) {
			var payload struct {
				ID    int64  `json:"id"`
				Email string `json:"email"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				return OAuthUserInfo{}, err
			}
			return OAuthUserInfo{Subject: fmt.Sprintf("%d", payload.ID), Email: payload.Email}, nil
		},
		httpClient: http.DefaultClient,
	}
}

func (client *httpOAuthClient) AuthCodeURL(state, codeChallenge string) string {
	return client.config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

func (client *httpOAuthClient) Exchange(ctx context.Context, code, codeVerifier string) (OAuthUserInfo, error) {
	token, err := client.config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return OAuthUserInfo{}, fmt.Errorf("exchange authorization code: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.userInfoURL, nil)
	if err != nil {
		return OAuthUserInfo{}, fmt.Errorf("build userinfo request: %w", err)
	}
	token.SetAuthHeader(request)
	request.Header.Set("Accept", "application/json")

	response, err := client.httpClient.Do(request)
	if err != nil {
		return OAuthUserInfo{}, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return OAuthUserInfo{}, fmt.Errorf("read userinfo response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return OAuthUserInfo{}, fmt.Errorf("userinfo request failed with status %d", response.StatusCode)
	}

	return client.extractInfo(body)
}
