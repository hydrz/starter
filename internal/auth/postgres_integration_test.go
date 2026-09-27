//go:build integration

package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestAuthAndMFAQueryMainPaths(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)
	user := createUser(t, ctx, db.Queries)

	if updated, err := db.Queries.MarkUserEmailVerified(ctx, user.ID); err != nil || updated != 1 {
		t.Fatalf("verify user = %d, %v", updated, err)
	}
	if updated, err := db.Queries.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{ID: user.ID, PasswordHash: "new-hash"}); err != nil || updated != 1 {
		t.Fatalf("update password = %d, %v", updated, err)
	}
	family, err := db.Queries.CreateRefreshTokenFamily(ctx, user.ID)
	if err != nil {
		t.Fatalf("create refresh family: %v", err)
	}
	digest := []byte("refresh-digest")
	if _, err := db.Queries.CreateRefreshSession(ctx, store.CreateRefreshSessionParams{FamilyID: family.ID, TokenDigest: digest, ExpiresAt: future()}); err != nil {
		t.Fatalf("create refresh session: %v", err)
	}
	if _, err := db.Queries.ConsumeRefreshSession(ctx, digest); err != nil {
		t.Fatalf("consume refresh session: %v", err)
	}
	if _, err := db.Queries.ConsumeRefreshSession(ctx, digest); err != pgx.ErrNoRows {
		t.Fatalf("reconsume refresh session = %v, want pgx.ErrNoRows", err)
	}
	reason := "test"
	if revoked, err := db.Queries.RevokeRefreshFamily(ctx, store.RevokeRefreshFamilyParams{ID: family.ID, RevokeReason: &reason}); err != nil || revoked != 1 {
		t.Fatalf("revoke refresh family = %d, %v", revoked, err)
	}

	oneTime, err := db.Queries.CreateOneTimeToken(ctx, store.CreateOneTimeTokenParams{UserID: user.ID, Purpose: "email_verification", TokenDigest: []byte("one-time"), ExpiresAt: future()})
	if err != nil {
		t.Fatalf("create one-time token: %v", err)
	}
	if _, err := db.Queries.ConsumeOneTimeToken(ctx, store.ConsumeOneTimeTokenParams{TokenDigest: []byte("one-time"), Purpose: "email_verification"}); err != nil {
		t.Fatalf("consume one-time token %s: %v", uuid.UUID(oneTime.ID.Bytes), err)
	}
	name := "test"
	key, err := db.Queries.CreateAPIKey(ctx, store.CreateAPIKeyParams{UserID: user.ID, Name: name, KeyPrefix: "sk_test", SecretDigest: []byte("api-key")})
	if err != nil {
		t.Fatalf("create API key: %v", err)
	}
	if _, err := db.Queries.GetActiveAPIKeyByDigest(ctx, []byte("api-key")); err != nil {
		t.Fatalf("get active API key: %v", err)
	}
	if revoked, err := db.Queries.RevokeAPIKeyForUser(ctx, store.RevokeAPIKeyForUserParams{ID: key.ID, UserID: user.ID}); err != nil || revoked != 1 {
		t.Fatalf("revoke API key = %d, %v", revoked, err)
	}

	otp, err := db.Queries.CreateEmailOTPChallenge(ctx, store.CreateEmailOTPChallengeParams{Email: user.Email, Purpose: "sign_in", CodeDigest: []byte("otp"), ExpiresAt: future()})
	if err != nil {
		t.Fatalf("create email OTP: %v", err)
	}
	if _, err := db.Queries.FindActiveEmailOTPChallenge(ctx, store.FindActiveEmailOTPChallengeParams{Email: user.Email, Purpose: "sign_in"}); err != nil {
		t.Fatalf("find active OTP: %v", err)
	}
	if _, err := db.Queries.IncrementEmailOTPAttempt(ctx, otp.ID); err != nil {
		t.Fatalf("increment OTP attempt: %v", err)
	}
	if _, err := db.Queries.ConsumeEmailOTPChallengeByID(ctx, otp.ID); err != nil {
		t.Fatalf("consume email OTP: %v", err)
	}

	factor, err := db.Queries.CreateTOTPFactor(ctx, store.CreateTOTPFactorParams{UserID: user.ID, SecretCiphertext: []byte("ciphertext"), SecretNonce: []byte("nonce")})
	if err != nil {
		t.Fatalf("create TOTP factor: %v", err)
	}
	if activated, err := db.Queries.ActivateTOTPFactor(ctx, store.ActivateTOTPFactorParams{ID: factor.ID, UserID: user.ID}); err != nil || activated != 1 {
		t.Fatalf("activate TOTP factor = %d, %v", activated, err)
	}
	if _, err := db.Queries.GetVerifiedTOTPFactorForUser(ctx, user.ID); err != nil {
		t.Fatalf("get verified TOTP factor: %v", err)
	}
	if err := db.Queries.CreateTOTPRecoveryCode(ctx, store.CreateTOTPRecoveryCodeParams{FactorID: factor.ID, CodeDigest: []byte("recovery")}); err != nil {
		t.Fatalf("create recovery code: %v", err)
	}
	if _, err := db.Queries.ConsumeTOTPRecoveryCode(ctx, store.ConsumeTOTPRecoveryCodeParams{FactorID: factor.ID, CodeDigest: []byte("recovery")}); err != nil {
		t.Fatalf("consume recovery code: %v", err)
	}
	challenge, err := db.Queries.CreateMFAChallenge(ctx, store.CreateMFAChallengeParams{UserID: user.ID, FactorID: factor.ID, ExpiresAt: future()})
	if err != nil {
		t.Fatalf("create MFA challenge: %v", err)
	}
	if _, err := db.Queries.ConsumeMFAChallenge(ctx, challenge.ID); err != nil {
		t.Fatalf("consume MFA challenge: %v", err)
	}
}

func createUser(t *testing.T, ctx context.Context, queries *store.Queries) store.User {
	t.Helper()
	user, err := queries.CreateUser(ctx, store.CreateUserParams{Email: uuid.NewString()[:12] + "@example.com", PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func future() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true}
}
