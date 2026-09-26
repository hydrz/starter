package auth_test

import (
	"context"
	"testing"

	"github.com/hydrz/starter/internal/api/authapi"
	"github.com/hydrz/starter/internal/auth"
)

func TestHTTPHandlerSignUpAndSignIn(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	handler := auth.NewHTTPHandler(h.service)
	ctx := context.Background()

	signUpRes, err := handler.SignUp(ctx, &authapi.SignUpInput{Email: "handler@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	identity, ok := signUpRes.(*authapi.Identity)
	if !ok {
		t.Fatalf("SignUp() result type = %T, want *authapi.Identity", signUpRes)
	}
	if identity.Email != "handler@example.com" {
		t.Errorf("identity.Email = %q, want handler@example.com", identity.Email)
	}

	signInRes, err := handler.PasswordSignIn(ctx, &authapi.PasswordSignInInput{Email: "handler@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("PasswordSignIn() error = %v", err)
	}
	token, ok := signInRes.(*authapi.AccessTokenResponse)
	if !ok {
		t.Fatalf("PasswordSignIn() result type = %T, want *authapi.AccessTokenResponse", signInRes)
	}
	if token.AccessToken == "" {
		t.Error("PasswordSignIn() returned empty access token")
	}
}

func TestHTTPHandlerSignUpRejectsDuplicateEmail(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	handler := auth.NewHTTPHandler(h.service)
	ctx := context.Background()

	if _, err := handler.SignUp(ctx, &authapi.SignUpInput{Email: "dup-handler@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("SignUp() first call error = %v", err)
	}
	res, err := handler.SignUp(ctx, &authapi.SignUpInput{Email: "dup-handler@example.com", Password: "correct-horse"})
	if err != nil {
		t.Fatalf("SignUp() second call error = %v", err)
	}
	if _, ok := res.(*authapi.SignUpConflict); !ok {
		t.Fatalf("SignUp() second call result type = %T, want *authapi.SignUpConflict", res)
	}
}

func TestHTTPHandlerPasswordSignInRejectsWrongPassword(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	handler := auth.NewHTTPHandler(h.service)
	ctx := context.Background()

	if _, err := handler.SignUp(ctx, &authapi.SignUpInput{Email: "wrong-handler@example.com", Password: "correct-horse"}); err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	res, err := handler.PasswordSignIn(ctx, &authapi.PasswordSignInInput{Email: "wrong-handler@example.com", Password: "incorrect-horse"})
	if err != nil {
		t.Fatalf("PasswordSignIn() error = %v", err)
	}
	if _, ok := res.(*authapi.PasswordSignInUnauthorized); !ok {
		t.Fatalf("PasswordSignIn() result type = %T, want *authapi.PasswordSignInUnauthorized", res)
	}
}

func TestHTTPHandlerGetCurrentIdentityRequiresPrincipal(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	handler := auth.NewHTTPHandler(h.service)

	res, err := handler.GetCurrentIdentity(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentIdentity() error = %v", err)
	}
	if _, ok := res.(*authapi.ApiError); !ok {
		t.Fatalf("GetCurrentIdentity() without principal result type = %T, want *authapi.ApiError", res)
	}
}
