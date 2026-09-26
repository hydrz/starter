package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-faster/jx"
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
		case errors.Is(err, ErrMFARequired):
			return toAPIMFAChallenge(tokens), nil
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

func toAPIMFAChallenge(tokens IssuedTokens) *authapi.MFAChallengeBody {
	return &authapi.MFAChallengeBody{
		MfaRequired: authapi.MFAChallengeBodyMfaRequiredTrue,
		ChallengeId: tokens.MFAChallengeID,
	}
}

// --- Email OTP --------------------------------------------------------

func (h *HTTPHandler) RequestEmailOTP(ctx context.Context, req *authapi.EmailOTPRequestInput) (authapi.RequestEmailOTPRes, error) {
	err := h.service.RequestEmailOTP(ctx, req.Email, RequestMetadataFromContext(ctx).IPAddress)
	if err != nil {
		if errors.Is(err, ErrRateLimited) {
			return &authapi.RequestEmailOTPTooManyRequests{Code: "rate_limited", Message: "Too many requests. Try again later."}, nil
		}
		return nil, err
	}
	return &authapi.RequestEmailOTPNoContent{}, nil
}

func (h *HTTPHandler) VerifyEmailOTP(ctx context.Context, req *authapi.EmailOTPVerifyInput) (authapi.VerifyEmailOTPRes, error) {
	tokens, err := h.service.VerifyEmailOTP(ctx, req.Email, req.Code, RequestMetadataFromContext(ctx))
	if err != nil {
		switch {
		case errors.Is(err, ErrMFARequired):
			return toAPIMFAChallenge(tokens), nil
		case errors.Is(err, ErrInvalidCredentials):
			return &authapi.VerifyEmailOTPUnauthorized{Code: "invalid_code", Message: "The code is invalid or expired."}, nil
		default:
			return nil, err
		}
	}
	SetRefreshCookie(ctx, tokens.RefreshToken, tokens.RefreshExpires)
	return toAPIAccessToken(tokens), nil
}

// --- TOTP enrollment and MFA challenges --------------------------------

func (h *HTTPHandler) BeginTOTPEnrollment(ctx context.Context) (authapi.BeginTOTPEnrollmentRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.ApiError{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	provisioning, err := h.service.BeginTOTPEnrollment(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &authapi.TOTPEnrollmentBeginResponse{
		FactorId:   provisioning.FactorID,
		Secret:     provisioning.Secret,
		OtpauthUrl: provisioning.URL,
	}, nil
}

func (h *HTTPHandler) ConfirmTOTPEnrollment(ctx context.Context, req *authapi.TOTPEnrollmentConfirmInput) (authapi.ConfirmTOTPEnrollmentRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.ConfirmTOTPEnrollmentUnauthorized{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	codes, err := h.service.ConfirmTOTPEnrollment(ctx, userID, req.FactorId, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, ErrTOTPEnrollmentNotFound), errors.Is(err, ErrTOTPInvalidCode):
			return &authapi.ConfirmTOTPEnrollmentBadRequest{Code: "invalid_code", Message: "The enrollment or code is invalid."}, nil
		default:
			return nil, err
		}
	}
	return &authapi.TOTPEnrollmentConfirmResponse{RecoveryCodes: codes}, nil
}

func (h *HTTPHandler) VerifyMFAChallenge(ctx context.Context, req *authapi.MFAChallengeVerifyInput) (authapi.VerifyMFAChallengeRes, error) {
	tokens, err := h.service.VerifyMFAChallenge(ctx, req.ChallengeId, req.Code, RequestMetadataFromContext(ctx))
	if err != nil {
		switch {
		case errors.Is(err, ErrMFAChallengeNotFound):
			return &authapi.VerifyMFAChallengeUnauthorized{Code: "invalid_challenge", Message: "The challenge is invalid or expired."}, nil
		case errors.Is(err, ErrTOTPInvalidCode):
			return &authapi.VerifyMFAChallengeBadRequest{Code: "invalid_code", Message: "The code is invalid."}, nil
		default:
			return nil, err
		}
	}
	SetRefreshCookie(ctx, tokens.RefreshToken, tokens.RefreshExpires)
	return toAPIAccessToken(tokens), nil
}

// --- OAuth --------------------------------------------------------------

func (h *HTTPHandler) BeginOAuthSignIn(ctx context.Context, params authapi.BeginOAuthSignInParams) (authapi.BeginOAuthSignInRes, error) {
	url, err := h.service.BeginOAuthSignIn(ctx, string(params.Provider))
	if err != nil {
		return &authapi.ApiError{Code: "unknown_provider", Message: "Unsupported OAuth provider."}, nil
	}
	return &authapi.OAuthAuthorizationResponse{AuthorizationUrl: url}, nil
}

func (h *HTTPHandler) CompleteOAuthSignIn(ctx context.Context, req *authapi.OAuthCallbackInput, params authapi.CompleteOAuthSignInParams) (authapi.CompleteOAuthSignInRes, error) {
	tokens, err := h.service.CompleteOAuthSignIn(ctx, string(params.Provider), req.State, req.Code, RequestMetadataFromContext(ctx))
	if err != nil {
		switch {
		case errors.Is(err, ErrMFARequired):
			return toAPIMFAChallenge(tokens), nil
		case errors.Is(err, ErrOAuthStateInvalid), errors.Is(err, ErrOAuthProviderUnknown):
			return &authapi.CompleteOAuthSignInBadRequest{Code: "invalid_oauth_callback", Message: "The OAuth callback is invalid or expired."}, nil
		default:
			return nil, err
		}
	}
	SetRefreshCookie(ctx, tokens.RefreshToken, tokens.RefreshExpires)
	return toAPIAccessToken(tokens), nil
}

func (h *HTTPHandler) BeginOAuthLink(ctx context.Context, params authapi.BeginOAuthLinkParams) (authapi.BeginOAuthLinkRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.BeginOAuthLinkUnauthorized{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	issuedAt, _ := AuthenticatedAtFromContext(ctx)
	url, err := h.service.BeginOAuthLink(ctx, string(params.Provider), userID, issuedAt)
	if err != nil {
		switch {
		case errors.Is(err, ErrReauthenticationRequired):
			return &authapi.BeginOAuthLinkForbidden{Code: "reauthentication_required", Message: "Sign in again before linking a provider."}, nil
		case errors.Is(err, ErrOAuthProviderUnknown):
			return &authapi.BeginOAuthLinkBadRequest{Code: "unknown_provider", Message: "Unsupported OAuth provider."}, nil
		default:
			return nil, err
		}
	}
	return &authapi.OAuthAuthorizationResponse{AuthorizationUrl: url}, nil
}

func (h *HTTPHandler) CompleteOAuthLink(ctx context.Context, req *authapi.OAuthCallbackInput, params authapi.CompleteOAuthLinkParams) (authapi.CompleteOAuthLinkRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.CompleteOAuthLinkUnauthorized{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	if err := h.service.LinkOAuthAccount(ctx, string(params.Provider), req.State, req.Code, userID); err != nil {
		switch {
		case errors.Is(err, ErrOAuthAccountTaken):
			return &authapi.CompleteOAuthLinkConflict{Code: "oauth_account_taken", Message: "That provider identity is already linked to another account."}, nil
		case errors.Is(err, ErrOAuthStateInvalid), errors.Is(err, ErrOAuthProviderUnknown):
			return &authapi.CompleteOAuthLinkBadRequest{Code: "invalid_oauth_callback", Message: "The OAuth callback is invalid or expired."}, nil
		default:
			return nil, err
		}
	}
	return &authapi.CompleteOAuthLinkNoContent{}, nil
}

func (h *HTTPHandler) ListOAuthAccounts(ctx context.Context) (authapi.ListOAuthAccountsRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.ApiError{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	accounts, err := h.service.ListOAuthAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make(authapi.ListOAuthAccountsOKApplicationJSON, len(accounts))
	for i, account := range accounts {
		item := authapi.OAuthAccount{
			Provider:  authapi.OAuthProvider(account.Provider),
			Subject:   account.Subject,
			CreatedAt: account.CreatedAt,
		}
		if account.Email != "" {
			item.Email = authapi.NewOptString(account.Email)
		}
		result[i] = item
	}
	return &result, nil
}

// --- WebAuthn -------------------------------------------------------------

func (h *HTTPHandler) BeginWebAuthnRegistration(ctx context.Context) (authapi.BeginWebAuthnRegistrationRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.ApiError{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	creation, err := h.service.BeginWebAuthnRegistration(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrWebAuthnUnauthenticated) {
			return &authapi.ApiError{Code: "unauthorized", Message: "Authentication is required."}, nil
		}
		return nil, err
	}
	publicKey, err := toRawMap(creation.Response)
	if err != nil {
		return nil, err
	}
	result := make(authapi.WebAuthnRegistrationBeginResponsePublicKey, len(publicKey))
	for key, value := range publicKey {
		result[key] = jx.Raw(value)
	}
	return &authapi.WebAuthnRegistrationBeginResponse{PublicKey: result}, nil
}

func (h *HTTPHandler) FinishWebAuthnRegistration(ctx context.Context, req *authapi.WebAuthnRegistrationFinishInput) (authapi.FinishWebAuthnRegistrationRes, error) {
	userID, ok := principalFromContext(ctx)
	if !ok {
		return &authapi.FinishWebAuthnRegistrationUnauthorized{Code: "unauthorized", Message: "Authentication is required."}, nil
	}
	credentialJSON, err := fromRawMap(map[string]json.RawMessage(rawMapFromCredential(req.Credential)))
	if err != nil {
		return &authapi.FinishWebAuthnRegistrationBadRequest{Code: "invalid_credential", Message: "The credential response is malformed."}, nil
	}
	if err := h.service.FinishWebAuthnRegistration(ctx, userID, credentialJSON); err != nil {
		switch {
		case errors.Is(err, ErrWebAuthnUnauthenticated):
			return &authapi.FinishWebAuthnRegistrationUnauthorized{Code: "unauthorized", Message: "Authentication is required."}, nil
		case errors.Is(err, ErrWebAuthnChallengeInvalid), errors.Is(err, ErrWebAuthnVerificationFailed):
			return &authapi.FinishWebAuthnRegistrationBadRequest{Code: "invalid_credential", Message: "The passkey could not be verified."}, nil
		default:
			return nil, err
		}
	}
	return &authapi.FinishWebAuthnRegistrationNoContent{}, nil
}

func (h *HTTPHandler) BeginWebAuthnAuthentication(ctx context.Context) (*authapi.WebAuthnAuthenticationBeginResponse, error) {
	assertion, err := h.service.BeginWebAuthnAuthentication(ctx)
	if err != nil {
		return nil, err
	}
	publicKey, err := toRawMap(assertion.Response)
	if err != nil {
		return nil, err
	}
	result := make(authapi.WebAuthnAuthenticationBeginResponsePublicKey, len(publicKey))
	for key, value := range publicKey {
		result[key] = jx.Raw(value)
	}
	return &authapi.WebAuthnAuthenticationBeginResponse{PublicKey: result}, nil
}

func (h *HTTPHandler) FinishWebAuthnAuthentication(ctx context.Context, req *authapi.WebAuthnAuthenticationFinishInput) (authapi.FinishWebAuthnAuthenticationRes, error) {
	assertionJSON, err := fromRawMap(map[string]json.RawMessage(rawMapFromAssertion(req.Credential)))
	if err != nil {
		return &authapi.FinishWebAuthnAuthenticationBadRequest{Code: "invalid_credential", Message: "The credential response is malformed."}, nil
	}
	tokens, err := h.service.FinishWebAuthnAuthentication(ctx, assertionJSON, RequestMetadataFromContext(ctx))
	if err != nil {
		switch {
		case errors.Is(err, ErrMFARequired):
			return toAPIMFAChallenge(tokens), nil
		case errors.Is(err, ErrWebAuthnChallengeInvalid), errors.Is(err, ErrWebAuthnVerificationFailed):
			return &authapi.FinishWebAuthnAuthenticationBadRequest{Code: "invalid_credential", Message: "The passkey could not be verified."}, nil
		default:
			return nil, err
		}
	}
	SetRefreshCookie(ctx, tokens.RefreshToken, tokens.RefreshExpires)
	return toAPIAccessToken(tokens), nil
}

// toRawMap marshals v (a webauthn protocol options struct) to JSON and
// re-decodes it as a field map, matching the ogen-generated
// map[string]jx.Raw "free-form object" schemas the TypeSpec Record<unknown>
// fields compile to.
func toRawMap(v any) (map[string]jsonRaw, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var decoded map[string]jsonRaw
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

// fromRawMap re-encodes a decoded field map back into a single JSON object,
// the shape github.com/go-webauthn/webauthn/protocol's parsers expect.
func fromRawMap(m map[string]json.RawMessage) ([]byte, error) {
	return json.Marshal(m)
}

// jsonRaw is an alias so toRawMap's map literal matches json.RawMessage's
// underlying []byte representation without importing authapi's per-schema
// named map types here.
type jsonRaw = json.RawMessage

func rawMapFromCredential(credential authapi.WebAuthnRegistrationFinishInputCredential) map[string]jsonRaw {
	result := make(map[string]jsonRaw, len(credential))
	for key, value := range credential {
		result[key] = json.RawMessage(value)
	}
	return result
}

func rawMapFromAssertion(credential authapi.WebAuthnAuthenticationFinishInputCredential) map[string]jsonRaw {
	result := make(map[string]jsonRaw, len(credential))
	for key, value := range credential {
		result[key] = json.RawMessage(value)
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
