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
}
