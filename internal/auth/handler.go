package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/hydrz/starter/internal/api/authapi"
)

// HTTPHandler adapts authapi.Handler (generated from
// packages/contracts/features/auth) to Service. It never leaks store/ogen
// types outside this file: everything below the handler boundary works
// with the auth package's own domain types.
type HTTPHandler struct {
	service *Service
}

func NewHTTPHandler(service *Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

var _ authapi.Handler = (*HTTPHandler)(nil)

func (h *HTTPHandler) SignUp(ctx context.Context, req *authapi.SignUpInput) (authapi.SignUpRes, error) {
	user, err := h.service.SignUp(ctx, SignUpInput{Email: req.Email, Password: req.Password})
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailTaken):
			return &authapi.SignUpConflict{Code: "email_taken", Message: "An account with that email already exists."}, nil
		case errors.Is(err, ErrInvalidInput):
			return &authapi.SignUpBadRequest{Code: "invalid_request", Message: err.Error()}, nil
		default:
			return nil, err
		}
	}
	identity := toAPIIdentity(user)
	return &identity, nil
}

func (h *HTTPHandler) PasswordSignIn(ctx context.Context, req *authapi.PasswordSignInInput) (authapi.PasswordSignInRes, error) {
	tokens, err := h.service.PasswordSignIn(ctx, PasswordSignInInput{Email: req.Email, Password: req.Password}, RequestMetadataFromContext(ctx))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			return &authapi.PasswordSignInUnauthorized{Code: "invalid_credentials", Message: "Email or password is incorrect."}, nil
		case errors.Is(err, ErrInvalidInput):
			return &authapi.PasswordSignInBadRequest{Code: "invalid_request", Message: err.Error()}, nil
		default:
			return nil, err
		}
	}
	SetRefreshCookie(ctx, tokens.RefreshToken, tokens.RefreshExpires)
	return toAPIAccessToken(tokens), nil
}

func (h *HTTPHandler) SignOut(ctx context.Context) (authapi.SignOutRes, error) {
	if rawToken, ok := RefreshTokenFromContext(ctx); ok {
		if err := h.service.SignOut(ctx, rawToken); err != nil {
			return nil, err
		}
	}
	ClearRefreshCookie(ctx)
	return &authapi.SignOutNoContent{}, nil
}

func (h *HTTPHandler) RefreshAccessToken(ctx context.Context) (authapi.RefreshAccessTokenRes, error) {
	rawToken, ok := RefreshTokenFromContext(ctx)
	if !ok {
		return &authapi.RefreshAccessTokenUnauthorized{Code: "missing_refresh_token", Message: "No refresh token was provided."}, nil
	}

	tokens, err := h.service.RefreshAccessToken(ctx, rawToken, RequestMetadataFromContext(ctx))
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionReused):
			ClearRefreshCookie(ctx)
			return &authapi.RefreshAccessTokenConflict{Code: "refresh_token_reused", Message: "This refresh token was already used; the session has been revoked."}, nil
		case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrInvalidInput):
			ClearRefreshCookie(ctx)
			return &authapi.RefreshAccessTokenUnauthorized{Code: "invalid_refresh_token", Message: "The refresh token is invalid or expired."}, nil
		default:
			return nil, err
		}
	}
	SetRefreshCookie(ctx, tokens.RefreshToken, tokens.RefreshExpires)
	return toAPIAccessToken(tokens), nil
}

func (h *HTTPHandler) GetCurrentIdentity(ctx context.Context) (authapi.GetCurrentIdentityRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.ApiError{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	user, err := h.service.CurrentIdentity(ctx, userID)
	if err != nil {
		return &authapi.ApiError{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	identity := toAPIIdentity(user)
	return &identity, nil
}

func (h *HTTPHandler) RequestEmailVerification(ctx context.Context, req *authapi.VerificationRequestInput) (authapi.RequestEmailVerificationRes, error) {
	if err := h.service.RequestEmailVerification(ctx, req.Email); err != nil {
		return &authapi.RequestEmailVerificationBadRequest{Code: "invalid_request", Message: "Invalid email address."}, nil
	}
	return &authapi.RequestEmailVerificationNoContent{}, nil
}

func (h *HTTPHandler) ConfirmEmailVerification(ctx context.Context, req *authapi.VerificationConfirmInput) (authapi.ConfirmEmailVerificationRes, error) {
	if err := h.service.ConfirmEmailVerification(ctx, req.Token); err != nil {
		return &authapi.ApiError{Code: "invalid_token", Message: "The verification token is invalid or expired."}, nil
	}
	return &authapi.ConfirmEmailVerificationNoContent{}, nil
}

func (h *HTTPHandler) RequestPasswordReset(ctx context.Context, req *authapi.VerificationRequestInput) (authapi.RequestPasswordResetRes, error) {
	if err := h.service.RequestPasswordReset(ctx, req.Email); err != nil {
		return &authapi.RequestPasswordResetBadRequest{Code: "invalid_request", Message: "Invalid email address."}, nil
	}
	return &authapi.RequestPasswordResetNoContent{}, nil
}

func (h *HTTPHandler) ConfirmPasswordReset(ctx context.Context, req *authapi.PasswordResetConfirmInput) (authapi.ConfirmPasswordResetRes, error) {
	if err := h.service.ConfirmPasswordReset(ctx, req.Token, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrTokenNotFound), errors.Is(err, ErrInvalidInput):
			return &authapi.ApiError{Code: "invalid_token", Message: "The reset token is invalid, expired, or the password does not meet requirements."}, nil
		default:
			return nil, err
		}
	}
	return &authapi.ConfirmPasswordResetNoContent{}, nil
}

func (h *HTTPHandler) ListSessions(ctx context.Context) (authapi.ListSessionsRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.ApiError{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	sessions, err := h.service.ListSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make(authapi.ListSessionsOKApplicationJSON, len(sessions))
	for i, session := range sessions {
		result[i] = toAPISession(session)
	}
	return &result, nil
}

func (h *HTTPHandler) RevokeSession(ctx context.Context, params authapi.RevokeSessionParams) (authapi.RevokeSessionRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.RevokeSessionUnauthorized{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	if err := h.service.RevokeSession(ctx, params.ID.String(), userID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return &authapi.RevokeSessionNotFound{Code: "not_found", Message: "Session was not found."}, nil
		}
		return nil, err
	}
	return &authapi.RevokeSessionNoContent{}, nil
}

func (h *HTTPHandler) CreateAPIKey(ctx context.Context, req *authapi.APIKeyCreateInput) (authapi.CreateAPIKeyRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.CreateAPIKeyUnauthorized{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	var expiresAt *time.Time
	if value, isSet := req.ExpiresAt.Get(); isSet {
		expiresAt = &value
	}
	created, err := h.service.CreateAPIKey(ctx, userID, req.Name, expiresAt)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return &authapi.CreateAPIKeyBadRequest{Code: "invalid_request", Message: "Invalid API key name."}, nil
		}
		return nil, err
	}
	return toAPICreatedKey(created), nil
}

func (h *HTTPHandler) ListAPIKeys(ctx context.Context) (authapi.ListAPIKeysRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.ApiError{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	keys, err := h.service.ListAPIKeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make(authapi.ListAPIKeysOKApplicationJSON, len(keys))
	for i, key := range keys {
		result[i] = toAPIKey(key)
	}
	return &result, nil
}

func (h *HTTPHandler) RevokeAPIKey(ctx context.Context, params authapi.RevokeAPIKeyParams) (authapi.RevokeAPIKeyRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.RevokeAPIKeyUnauthorized{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	if err := h.service.RevokeAPIKey(ctx, params.ID.String(), userID); err != nil {
		if errors.Is(err, ErrAPIKeyNotFound) {
			return &authapi.RevokeAPIKeyNotFound{Code: "not_found", Message: "API key was not found."}, nil
		}
		return nil, err
	}
	return &authapi.RevokeAPIKeyNoContent{}, nil
}

func toAPIIdentity(user User) authapi.Identity {
	return authapi.Identity{
		ID:            uuid.MustParse(user.ID),
		Email:         user.Email,
		EmailVerified: user.EmailVerifiedAt != nil,
		CreatedAt:     user.CreatedAt,
	}
}

func toAPIAccessToken(tokens IssuedTokens) *authapi.AccessTokenResponse {
	return &authapi.AccessTokenResponse{
		AccessToken: tokens.AccessToken,
		TokenType:   authapi.AccessTokenResponseTokenTypeBearer,
		ExpiresAt:   tokens.AccessExpiresAt,
	}
}

func toAPISession(session RefreshSession) authapi.Session {
	result := authapi.Session{
		ID:        uuid.MustParse(session.ID),
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
	if session.LastUsedAt != nil {
		result.LastUsedAt = authapi.NewOptDateTime(*session.LastUsedAt)
	}
	if session.UserAgent != "" {
		result.UserAgent = authapi.NewOptString(session.UserAgent)
	}
	if session.IPAddress != "" {
		result.IpAddress = authapi.NewOptString(session.IPAddress)
	}
	return result
}

func toAPIKey(key APIKey) authapi.APIKey {
	result := authapi.APIKey{
		ID:        uuid.MustParse(key.ID),
		Name:      key.Name,
		Prefix:    key.KeyPrefix,
		CreatedAt: key.CreatedAt,
	}
	if key.ExpiresAt != nil {
		result.ExpiresAt = authapi.NewOptDateTime(*key.ExpiresAt)
	}
	if key.RevokedAt != nil {
		result.RevokedAt = authapi.NewOptDateTime(*key.RevokedAt)
	}
	if key.LastUsedAt != nil {
		result.LastUsedAt = authapi.NewOptDateTime(*key.LastUsedAt)
	}
	return result
}

func toAPICreatedKey(created CreatedAPIKey) *authapi.CreatedAPIKey {
	result := &authapi.CreatedAPIKey{
		ID:        uuid.MustParse(created.ID),
		Name:      created.Name,
		Prefix:    created.KeyPrefix,
		CreatedAt: created.CreatedAt,
		Key:       created.Key,
	}
	if created.ExpiresAt != nil {
		result.ExpiresAt = authapi.NewOptDateTime(*created.ExpiresAt)
	}
	return result
}
