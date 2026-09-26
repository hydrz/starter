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
