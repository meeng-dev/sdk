/** What an error answer says, as useSend's SDKs read it. */
export interface ErrorResponse {
  /** The API's code, such as `NOT_UNIQUE` or `FORBIDDEN`; `INTERNAL_SERVER_ERROR` when the answer had none. */
  code: string;
  message: string;
  /** The HTTP status of the answer. */
  status?: number;
}

/** What every call answers: the data, or the error, never both. */
export type Result<T> = { data: T; error: null } | { data: null; error: ErrorResponse };

/** Options for a call that takes an idempotency key. */
export interface RequestOptions {
  /**
   * Makes a send safe to retry: the same key with the same body answers the
   * first result again; with another body, a 409 `NOT_UNIQUE`. The key is up to
   * 256 characters and expires after 24 hours.
   */
  idempotencyKey?: string;
  /** Cancels the call. */
  signal?: AbortSignal;
}

/** Options for any other call. */
export interface CallOptions {
  /** Cancels the call. */
  signal?: AbortSignal;
}

/**
 * A query string's values; undefined ones are left out. A list is sent once
 * for each value, as name=a&name=b.
 */
export type Query = Record<string, string | number | string[] | undefined>;

export interface RequestSpec {
  query?: Query;
  body?: unknown;
  options?: (RequestOptions & CallOptions) | undefined;
}

/** What a service calls the API through. */
export interface Requester {
  request<T>(method: string, path: string, spec?: RequestSpec): Promise<Result<T>>;
}
