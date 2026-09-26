import { describe, expect, it } from "vitest";
import {
  base64UrlToBuffer,
  bufferToBase64Url,
  decodeCreationOptions,
} from "./webauthn";

function bytesOf(...values: Array<number>): Uint8Array {
  return new Uint8Array(values);
}

describe("base64url <-> ArrayBuffer round-trip", () => {
  it("round-trips arbitrary byte sequences, including ones that need padding", () => {
    // Lengths 0..8 exercise every possible base64 padding case (mod-3 remainder 0/1/2).
    for (let length = 0; length <= 8; length++) {
      const bytes = bytesOf(
        ...Array.from({ length }, (_, index) => (index * 37) % 256),
      );
      const encoded = bufferToBase64Url(bytes);
      const decoded = new Uint8Array(base64UrlToBuffer(encoded));
      expect(Array.from(decoded)).toEqual(Array.from(bytes));
    }
  });

  it("never emits standard-base64 characters ('+', '/') or padding ('=')", () => {
    // Bytes chosen so the standard-base64 alphabet would need + and /.
    const bytes = bytesOf(0xfb, 0xff, 0xbf, 0x3e, 0x3f);
    const encoded = bufferToBase64Url(bytes);
    expect(encoded).not.toMatch(/[+/=]/);
  });

  it("decodes a known base64url string (no padding) to the expected bytes", () => {
    // "hello" in base64url.
    expect(new Uint8Array(base64UrlToBuffer("aGVsbG8"))).toEqual(
      bytesOf(104, 101, 108, 108, 111),
    );
  });

  it("accepts base64url strings that decode to a length requiring padding", () => {
    // 1-byte input's base64 form normally carries "==" padding, which must
    // never be emitted (or expected) in the base64url wire format.
    const oneByte = bytesOf(0xff);
    const encoded = bufferToBase64Url(oneByte);
    expect(encoded.endsWith("=")).toBe(false);
    expect(new Uint8Array(base64UrlToBuffer(encoded))).toEqual(oneByte);
  });
});

describe("decodeCreationOptions", () => {
  it("decodes challenge, user.id and excludeCredentials[].id from base64url into ArrayBuffers", () => {
    const challengeBytes = bytesOf(1, 2, 3, 4, 5);
    const userIdBytes = bytesOf(9, 8, 7);
    const excludedIdBytes = bytesOf(42, 43);

    const options = decodeCreationOptions({
      rp: { id: "example.com", name: "Example" },
      user: {
        id: bufferToBase64Url(userIdBytes),
        name: "user@example.com",
        displayName: "User",
      },
      challenge: bufferToBase64Url(challengeBytes),
      pubKeyCredParams: [{ type: "public-key", alg: -7 }],
      excludeCredentials: [
        {
          id: bufferToBase64Url(excludedIdBytes),
          type: "public-key",
        },
      ],
    });

    const publicKey = options.publicKey;
    expect(publicKey).toBeDefined();
    expect(new Uint8Array(publicKey!.challenge as ArrayBuffer)).toEqual(
      challengeBytes,
    );
    expect(new Uint8Array(publicKey!.user.id as ArrayBuffer)).toEqual(
      userIdBytes,
    );
    expect(
      new Uint8Array(publicKey!.excludeCredentials![0].id as ArrayBuffer),
    ).toEqual(excludedIdBytes);
  });
});
