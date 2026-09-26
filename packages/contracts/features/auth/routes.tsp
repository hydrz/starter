import "@typespec/http";
import "@typespec/openapi";
import "../../common/responses.tsp";
import "./models.tsp";

using TypeSpec.OpenAPI;
using Starter.Common;

namespace Starter.Auth;

@TypeSpec.Http.route("/api/auth")
@tag("Auth")
interface Auth {
  @TypeSpec.Http.post
  @TypeSpec.Http.route("/sign-up")
  @operationId("signUp")
  @summary("Create an account")
  signUp(@TypeSpec.Http.body input: SignUpInput):
    | CreatedResponse<Identity>
    | BadRequestResponse
    | ConflictResponse
    | TooManyRequestsResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/sign-in")
  @operationId("passwordSignIn")
  @summary("Sign in with email and password")
  passwordSignIn(@TypeSpec.Http.body input: PasswordSignInInput):
    | AccessTokenResponse
    | MFAChallengeResponse
    | BadRequestResponse
    | UnauthorizedResponse
    | TooManyRequestsResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/sign-out")
  @operationId("signOut")
  @summary("Sign out the current refresh session")
  signOut(): NoContentResponse | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/refresh")
  @operationId("refreshAccessToken")
  @summary("Rotate the refresh session and issue an access token")
  refreshAccessToken():
    | AccessTokenResponse
    | UnauthorizedResponse
    | ConflictResponse;

  @TypeSpec.Http.get
  @TypeSpec.Http.route("/me")
  @operationId("getCurrentIdentity")
  @summary("Get the current identity")
  getCurrentIdentity(): Identity | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/verification/request")
  @operationId("requestEmailVerification")
  @summary("Request an email verification message")
  requestEmailVerification(
    @TypeSpec.Http.body input: VerificationRequestInput,
  ): NoContentResponse | BadRequestResponse | TooManyRequestsResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/verification/confirm")
  @operationId("confirmEmailVerification")
  @summary("Confirm an email verification token")
  confirmEmailVerification(
    @TypeSpec.Http.body input: VerificationConfirmInput,
  ): NoContentResponse | BadRequestResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/password-reset/request")
  @operationId("requestPasswordReset")
  @summary("Request a password reset message")
  requestPasswordReset(@TypeSpec.Http.body input: VerificationRequestInput):
    | NoContentResponse
    | BadRequestResponse
    | TooManyRequestsResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/password-reset/confirm")
  @operationId("confirmPasswordReset")
  @summary("Confirm a password reset token")
  confirmPasswordReset(@TypeSpec.Http.body input: PasswordResetConfirmInput):
    | NoContentResponse
    | BadRequestResponse;

  @TypeSpec.Http.get
  @TypeSpec.Http.route("/sessions")
  @operationId("listSessions")
  @summary("List active refresh sessions")
  listSessions(): Session[] | UnauthorizedResponse;

  @TypeSpec.Http.delete
  @TypeSpec.Http.route("/sessions/{id}")
  @operationId("revokeSession")
  @summary("Revoke a refresh session")
  revokeSession(@TypeSpec.Http.path @format("uuid") id: string):
    | NoContentResponse
    | UnauthorizedResponse
    | NotFoundResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/api-keys")
  @operationId("createAPIKey")
  @summary("Create an API key")
  createAPIKey(@TypeSpec.Http.body input: APIKeyCreateInput):
    | CreatedResponse<CreatedAPIKey>
    | BadRequestResponse
    | UnauthorizedResponse;

  @TypeSpec.Http.get
  @TypeSpec.Http.route("/api-keys")
  @operationId("listAPIKeys")
  @summary("List API keys")
  listAPIKeys(): APIKey[] | UnauthorizedResponse;

  @TypeSpec.Http.delete
  @TypeSpec.Http.route("/api-keys/{id}")
  @operationId("revokeAPIKey")
  @summary("Revoke an API key")
  revokeAPIKey(@TypeSpec.Http.path @format("uuid") id: string):
    | NoContentResponse
    | UnauthorizedResponse
    | NotFoundResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/otp/request")
  @operationId("requestEmailOTP")
  @summary("Request an email one-time sign-in code")
  requestEmailOTP(@TypeSpec.Http.body input: EmailOTPRequestInput):
    | NoContentResponse
    | BadRequestResponse
    | TooManyRequestsResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/otp/verify")
  @operationId("verifyEmailOTP")
  @summary("Verify an email one-time sign-in code and complete sign-in")
  verifyEmailOTP(@TypeSpec.Http.body input: EmailOTPVerifyInput):
    | AccessTokenResponse
    | MFAChallengeResponse
    | BadRequestResponse
    | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/mfa/totp/enroll/begin")
  @operationId("beginTOTPEnrollment")
  @summary("Begin enrolling a TOTP authenticator app")
  beginTOTPEnrollment(): TOTPEnrollmentBeginResponse | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/mfa/totp/enroll/confirm")
  @operationId("confirmTOTPEnrollment")
  @summary("Confirm a TOTP enrollment and receive recovery codes once")
  confirmTOTPEnrollment(@TypeSpec.Http.body input: TOTPEnrollmentConfirmInput):
    | TOTPEnrollmentConfirmResponse
    | BadRequestResponse
    | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/mfa/challenge/verify")
  @operationId("verifyMFAChallenge")
  @summary("Complete sign-in with a TOTP or recovery code")
  verifyMFAChallenge(@TypeSpec.Http.body input: MFAChallengeVerifyInput):
    | AccessTokenResponse
    | BadRequestResponse
    | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/oauth/{provider}/begin")
  @operationId("beginOAuthSignIn")
  @summary("Begin an OAuth sign-in with Google or GitHub")
  beginOAuthSignIn(@TypeSpec.Http.path provider: OAuthProvider):
    | OAuthAuthorizationResponse
    | BadRequestResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/oauth/{provider}/callback")
  @operationId("completeOAuthSignIn")
  @summary("Complete an OAuth sign-in callback")
  completeOAuthSignIn(
    @TypeSpec.Http.path provider: OAuthProvider,
    @TypeSpec.Http.body input: OAuthCallbackInput,
  ):
    | AccessTokenResponse
    | MFAChallengeResponse
    | BadRequestResponse
    | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/oauth/{provider}/link/begin")
  @operationId("beginOAuthLink")
  @summary("Begin linking an OAuth provider to the current, recently-authenticated account")
  beginOAuthLink(@TypeSpec.Http.path provider: OAuthProvider):
    | OAuthAuthorizationResponse
    | BadRequestResponse
    | UnauthorizedResponse
    | ForbiddenResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/oauth/{provider}/link/callback")
  @operationId("completeOAuthLink")
  @summary("Complete linking an OAuth provider callback")
  completeOAuthLink(
    @TypeSpec.Http.path provider: OAuthProvider,
    @TypeSpec.Http.body input: OAuthCallbackInput,
  ):
    | NoContentResponse
    | BadRequestResponse
    | UnauthorizedResponse
    | ConflictResponse;

  @TypeSpec.Http.get
  @TypeSpec.Http.route("/oauth/accounts")
  @operationId("listOAuthAccounts")
  @summary("List OAuth providers linked to the current account")
  listOAuthAccounts(): OAuthAccount[] | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/webauthn/registration/begin")
  @operationId("beginWebAuthnRegistration")
  @summary("Begin registering a passkey for the current account")
  beginWebAuthnRegistration():
    | WebAuthnRegistrationBeginResponse
    | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/webauthn/registration/finish")
  @operationId("finishWebAuthnRegistration")
  @summary("Finish registering a passkey for the current account")
  finishWebAuthnRegistration(
    @TypeSpec.Http.body input: WebAuthnRegistrationFinishInput,
  ): NoContentResponse | BadRequestResponse | UnauthorizedResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/webauthn/authentication/begin")
  @operationId("beginWebAuthnAuthentication")
  @summary("Begin a discoverable (usernameless) passkey sign-in")
  beginWebAuthnAuthentication(): WebAuthnAuthenticationBeginResponse;

  @TypeSpec.Http.post
  @TypeSpec.Http.route("/webauthn/authentication/finish")
  @operationId("finishWebAuthnAuthentication")
  @summary("Finish a passkey sign-in and complete sign-in")
  finishWebAuthnAuthentication(
    @TypeSpec.Http.body input: WebAuthnAuthenticationFinishInput,
  ):
    | AccessTokenResponse
    | MFAChallengeResponse
    | BadRequestResponse
    | UnauthorizedResponse;
}
