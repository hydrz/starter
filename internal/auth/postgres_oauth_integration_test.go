//go:build integration

package auth_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hydrz/starter/internal/platform/testdb"
	"github.com/hydrz/starter/internal/store"
)

func TestOAuthQueriesMainPaths(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)
	user := createUser(t, ctx, db.Queries)

	state, err := db.Queries.CreateOAuthAuthorizationState(ctx, store.CreateOAuthAuthorizationStateParams{
		Provider: "github", StateDigest: []byte("oauth-state"), Nonce: "nonce",
		CodeVerifier: "verifier", Intent: "sign_in", ExpiresAt: future(),
	})
	if err != nil {
		t.Fatalf("create OAuth state: %v", err)
	}
	if state.ID.Valid == false {
		t.Fatal("created OAuth state has no id")
	}
	consumed, err := db.Queries.ConsumeOAuthAuthorizationState(ctx, store.ConsumeOAuthAuthorizationStateParams{
		StateDigest: []byte("oauth-state"), Provider: "github",
	})
	if err != nil || consumed.Intent != "sign_in" {
		t.Fatalf("consume OAuth state = %#v, %v", consumed, err)
	}
	if again, err := db.Queries.ConsumeOAuthAuthorizationState(ctx, store.ConsumeOAuthAuthorizationStateParams{
		StateDigest: []byte("oauth-state"), Provider: "github",
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("re-consume OAuth state = %#v, %v; want pgx.ErrNoRows", again, err)
	}

	accountEmail := user.Email
	account, err := db.Queries.CreateOAuthAccount(ctx, store.CreateOAuthAccountParams{
		UserID: user.ID, Provider: "github", ProviderSubject: "subject-" + uuid.NewString()[:8], Email: &accountEmail,
	})
	if err != nil {
		t.Fatalf("create OAuth account: %v", err)
	}
	if found, err := db.Queries.FindOAuthAccount(ctx, store.FindOAuthAccountParams{
		Provider: "github", ProviderSubject: account.ProviderSubject,
	}); err != nil || found.ID != account.ID {
		t.Fatalf("find OAuth account = %#v, %v", found, err)
	}
	if listed, err := db.Queries.ListOAuthAccountsForUser(ctx, user.ID); err != nil || len(listed) != 1 {
		t.Fatalf("list OAuth accounts = %d items, %v", len(listed), err)
	}
}

func TestWebAuthnQueriesMainPaths(t *testing.T) {
	t.Parallel()
	db := testdb.New(t)
	ctx := testdb.Context(t)
	user := createUser(t, ctx, db.Queries)

	challenge := []byte("webauthn-challenge")
	if _, err := db.Queries.CreateWebAuthnChallenge(ctx, store.CreateWebAuthnChallengeParams{
		Ceremony: "registration", UserID: user.ID, Challenge: challenge,
		SessionData: []byte(`{"challenge":"webauthn-challenge"}`), ExpiresAt: future(),
	}); err != nil {
		t.Fatalf("create WebAuthn challenge: %v", err)
	}
	if consumed, err := db.Queries.ConsumeWebAuthnChallenge(ctx, store.ConsumeWebAuthnChallengeParams{
		Challenge: challenge, Ceremony: "registration",
	}); err != nil || consumed.ID.Valid == false {
		t.Fatalf("consume WebAuthn challenge = %#v, %v", consumed, err)
	}

	credentialID := []byte("credential-id")
	userHandle := []byte("user-handle")
	created, err := db.Queries.CreateWebAuthnCredential(ctx, store.CreateWebAuthnCredentialParams{
		UserID: user.ID, CredentialID: credentialID, PublicKey: []byte("public-key"),
		AttestationType: "none", Aaguid: []byte{}, SignCount: 5, UserHandle: userHandle,
	})
	if err != nil {
		t.Fatalf("create WebAuthn credential: %v", err)
	}
	if fetched, err := db.Queries.GetWebAuthnCredentialByCredentialID(ctx, credentialID); err != nil || fetched.ID != created.ID {
		t.Fatalf("get credential by id = %#v, %v", fetched, err)
	}
	if updated, err := db.Queries.UpdateWebAuthnCredentialSignCount(ctx, store.UpdateWebAuthnCredentialSignCountParams{
		ID: created.ID, SignCount: 6,
	}); err != nil || updated != 1 {
		t.Fatalf("update sign count = %d, %v", updated, err)
	}
	if stale, err := db.Queries.UpdateWebAuthnCredentialSignCount(ctx, store.UpdateWebAuthnCredentialSignCountParams{
		ID: created.ID, SignCount: 3,
	}); err != nil || stale != 0 {
		t.Fatalf("stale sign count update = %d, %v; want 0 rows", stale, err)
	}
	if listed, err := db.Queries.ListWebAuthnCredentialsForUser(ctx, user.ID); err != nil || len(listed) != 1 {
		t.Fatalf("list WebAuthn credentials = %d items, %v", len(listed), err)
	}
}
