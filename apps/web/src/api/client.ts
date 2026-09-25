export interface ApiErrorPayload {
  code: string;
  message: string;
}

export class HttpError<T = ApiErrorPayload> extends Error {
  readonly status: number;
  readonly data: T;

  constructor(status: number, data: T, message?: string) {
    const displayMessage =
      message ??
      (typeof data === "object" && data && "message" in data
        ? String((data as { message: unknown }).message)
        : `HTTP request failed with status ${status}`);
    super(displayMessage);
    this.name = "HttpError";
    this.status = status;
    this.data = data;
  }
}

export const customClient = async <T>(
  url: string,
  options?: RequestInit,
): Promise<T> => {
  const res = await fetch(url, options);

  const isNoContent = [204, 205, 304].includes(res.status);
  const text = isNoContent ? null : await res.text();
  let bodyData: unknown = null;
  if (text) {
    try {
      bodyData = JSON.parse(text);
    } catch {
      bodyData = text;
    }
  }

  if (!res.ok) {
    throw new HttpError(res.status, bodyData as ApiErrorPayload);
  }

  return {
    data: bodyData,
    status: res.status,
    headers: res.headers,
  } as T;
};
