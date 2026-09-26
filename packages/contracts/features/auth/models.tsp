import "@typespec/http";
import "@typespec/openapi";

namespace Starter.Auth;

@friendlyName("SignUpInput")
model SignUpInput {
  @format("email")
  email: string;

  @minLength(8)
  @maxLength(128)
  password: string;
}

@friendlyName("PasswordSignInInput")
model PasswordSignInInput {
  @format("email")
  email: string;

  @minLength(1)
  password: string;
}

@friendlyName("AccessTokenResponse")
model AccessTokenResponse {
  accessToken: string;
  tokenType: "Bearer";
  expiresAt: utcDateTime;
}

@friendlyName("Identity")
model Identity {
  @format("uuid")
  id: string;

  @format("email")
  email: string;

  emailVerified: boolean;
  createdAt: utcDateTime;
}

@friendlyName("VerificationRequestInput")
model VerificationRequestInput {
  @format("email")
  email: string;
}

@friendlyName("VerificationConfirmInput")
model VerificationConfirmInput {
  token: string;
}

@friendlyName("PasswordResetConfirmInput")
model PasswordResetConfirmInput {
  token: string;

  @minLength(8)
  @maxLength(128)
  password: string;
}

@friendlyName("Session")
model Session {
  @format("uuid")
  id: string;

  createdAt: utcDateTime;
  lastUsedAt?: utcDateTime;
  expiresAt: utcDateTime;
  userAgent?: string;
  ipAddress?: string;
}

@friendlyName("APIKeyCreateInput")
model APIKeyCreateInput {
  @minLength(1)
  @maxLength(120)
  name: string;

  expiresAt?: utcDateTime;
}

@friendlyName("APIKey")
model APIKey {
  @format("uuid")
  id: string;

  name: string;
  prefix: string;
  expiresAt?: utcDateTime;
  revokedAt?: utcDateTime;
  lastUsedAt?: utcDateTime;
  createdAt: utcDateTime;
}

@friendlyName("CreatedAPIKey")
model CreatedAPIKey extends APIKey {
  key: string;
}

// A 202 (not 200) distinguishes "primary credential accepted, a second
// factor is still required" from a completed sign-in at the same operation,
// so ogen can generate distinct response variants instead of an ambiguous
// same-status anyOf.
@friendlyName("MFAChallengeResponse")
model MFAChallengeResponse {
  @TypeSpec.Http.statusCode statusCode: 202;
  @TypeSpec.Http.body body: MFAChallengeBody;
}

@friendlyName("MFAChallengeBody")
model MFAChallengeBody {
  mfaRequired: true;
  challengeId: string;
}

@friendlyName("EmailOTPRequestInput")
model EmailOTPRequestInput {
  @format("email")
  email: string;
}

@friendlyName("EmailOTPVerifyInput")
model EmailOTPVerifyInput {
  @format("email")
  email: string;

  @minLength(6)
  @maxLength(6)
  code: string;
}

@friendlyName("TOTPEnrollmentBeginResponse")
model TOTPEnrollmentBeginResponse {
  factorId: string;
  secret: string;
  otpauthUrl: string;
}

@friendlyName("TOTPEnrollmentConfirmInput")
model TOTPEnrollmentConfirmInput {
  factorId: string;

  @minLength(6)
  @maxLength(6)
  code: string;
}

@friendlyName("TOTPEnrollmentConfirmResponse")
model TOTPEnrollmentConfirmResponse {
  recoveryCodes: string[];
}

@friendlyName("MFAChallengeVerifyInput")
model MFAChallengeVerifyInput {
  challengeId: string;

  @minLength(1)
  code: string;
}

@friendlyName("OAuthProvider")
enum OAuthProvider {
  google,
  github,
}

@friendlyName("OAuthAuthorizationResponse")
model OAuthAuthorizationResponse {
  authorizationUrl: string;
}

@friendlyName("OAuthCallbackInput")
model OAuthCallbackInput {
  state: string;
  code: string;
}

@friendlyName("OAuthAccount")
model OAuthAccount {
  provider: OAuthProvider;
  subject: string;
  email?: string;
  createdAt: utcDateTime;
}

@friendlyName("WebAuthnRegistrationBeginResponse")
model WebAuthnRegistrationBeginResponse {
  /** The raw PublicKeyCredentialCreationOptions JSON for navigator.credentials.create(). */
  publicKey: Record<unknown>;
}

@friendlyName("WebAuthnRegistrationFinishInput")
model WebAuthnRegistrationFinishInput {
  /** The raw PublicKeyCredential JSON from navigator.credentials.create(). */
  credential: Record<unknown>;
}

@friendlyName("WebAuthnAuthenticationBeginResponse")
model WebAuthnAuthenticationBeginResponse {
  /** The raw PublicKeyCredentialRequestOptions JSON for navigator.credentials.get(). */
  publicKey: Record<unknown>;
}

@friendlyName("WebAuthnAuthenticationFinishInput")
model WebAuthnAuthenticationFinishInput {
  /** The raw PublicKeyCredential JSON from navigator.credentials.get(). */
  credential: Record<unknown>;
}
