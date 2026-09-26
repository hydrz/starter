/**
 * WebAuthn JSON <-> browser-API encoding helpers (DESIGN.md §6.2 passkeys).
 *
 * The backend's generated model types (`WebAuthnRegistrationBeginResponse`/
 * `WebAuthnRegistrationFinishInput`) carry the raw `PublicKeyCredentialCreationOptions`
 * / `PublicKeyCredential` JSON as untyped `{[key: string]: unknown}` blobs
 * (see `apps/web/src/api/generated/model/webAuthnRegistrationBeginResponsePublicKey.ts`)
 * because JSON transport can't carry `ArrayBuffer`s — `challenge`, `user.id`
 * and credential ids travel as base64url strings, while
 * `navigator.credentials.create()` requires real `ArrayBuffer`/`Uint8Array`
 * values. These helpers do that conversion in both directions.
 *
 * base64url (RFC 4648 §5), not standard base64: `-`/`_` instead of `+`/`/`,
 * no padding. Getting this wrong (using `atob`/`btoa` directly, or leaving
 * padding in) fails silently — the browser either rejects the credential
 * options or the server rejects the finish payload — so this module is
 * covered by a round-trip unit test (`webauthn.test.ts`).
 */

/** Decodes a base64url string (no padding) into an ArrayBuffer. */
export function base64UrlToBuffer(value: string): ArrayBuffer {
  const base64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const padLength = (4 - (base64.length % 4)) % 4;
  const padded = base64 + "=".repeat(padLength);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/** Encodes an ArrayBuffer/TypedArray into an unpadded base64url string. */
export function bufferToBase64Url(
  buffer: ArrayBuffer | ArrayBufferView,
): string {
  const bytes =
    buffer instanceof Uint8Array
      ? buffer
      : new Uint8Array(
          "buffer" in buffer ? buffer.buffer : (buffer as ArrayBuffer),
        );
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  const base64 = btoa(binary);
  return base64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/**
 * Shape of the raw JSON the server sends for
 * `WebAuthnRegistrationBeginResponse.publicKey` (a server-side serialization
 * of `PublicKeyCredentialCreationOptions` with buffers as base64url
 * strings). Only the fields this UI actually reads/passes through are
 * typed; everything else round-trips via the index signature.
 */
export interface PublicKeyCredentialCreationOptionsJSON {
  rp: PublicKeyCredentialRpEntity;
  user: { id: string; name: string; displayName: string };
  challenge: string;
  pubKeyCredParams: Array<PublicKeyCredentialParameters>;
  timeout?: number;
  excludeCredentials?: Array<{
    id: string;
    type: "public-key";
    transports?: Array<AuthenticatorTransport>;
  }>;
  authenticatorSelection?: AuthenticatorSelectionCriteria;
  attestation?: AttestationConveyancePreference;
  [key: string]: unknown;
}

/**
 * Converts the server's JSON creation options into the real
 * `CredentialCreationOptions` `navigator.credentials.create()` expects,
 * decoding `challenge`, `user.id` and every `excludeCredentials[].id` from
 * base64url into `ArrayBuffer`s.
 */
export function decodeCreationOptions(
  publicKey: Record<string, unknown>,
): CredentialCreationOptions {
  const json = publicKey as unknown as PublicKeyCredentialCreationOptionsJSON;
  return {
    publicKey: {
      ...json,
      challenge: base64UrlToBuffer(json.challenge),
      user: {
        ...json.user,
        id: base64UrlToBuffer(json.user.id),
      },
      excludeCredentials: json.excludeCredentials?.map((credential) => ({
        ...credential,
        id: base64UrlToBuffer(credential.id),
      })),
    },
  };
}

/**
 * Converts the `PublicKeyCredential` returned by
 * `navigator.credentials.create()` back into the plain-JSON shape the
 * server's `WebAuthnRegistrationFinishInput.credential` expects, encoding
 * `rawId`/`clientDataJSON`/`attestationObject` as base64url strings.
 */
export function encodeRegistrationCredential(
  credential: PublicKeyCredential,
): Record<string, unknown> {
  const response = credential.response as AuthenticatorAttestationResponse;
  return {
    id: credential.id,
    rawId: bufferToBase64Url(credential.rawId),
    type: credential.type,
    response: {
      clientDataJSON: bufferToBase64Url(response.clientDataJSON),
      attestationObject: bufferToBase64Url(response.attestationObject),
      transports:
        typeof response.getTransports === "function"
          ? response.getTransports()
          : undefined,
    },
    clientExtensionResults: credential.getClientExtensionResults(),
  };
}
