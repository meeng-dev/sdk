from __future__ import annotations

from typing import Any, Literal, TypedDict


class ErrorResponse(TypedDict):
    """What an error answer says, as useSend's SDKs read it."""

    code: str
    message: str


class MeengHTTPError(Exception):
    """Raised when the API refuses a request and ``raise_on_error`` is true (the default)."""

    def __init__(self, status_code: int, error: ErrorResponse, method: str, path: str) -> None:
        self.status_code = status_code
        self.error = error
        self.method = method
        self.path = path
        super().__init__(f"{method} {path} -> {status_code} {error['code']}: {error['message']}")


def error_of(status: int, reason: str, raw: bytes) -> ErrorResponse:
    """The error a refused answer carries: ``{"error": {"code", "message"}}``, ``{"error": "text"}`` or the error itself."""
    import json

    error: ErrorResponse = {"code": "INTERNAL_SERVER_ERROR", "message": reason}
    try:
        body: Any = json.loads(raw)
    except ValueError:
        return error
    if not isinstance(body, dict):
        return error
    found: Any = body.get("error", body)
    if isinstance(found, str):
        found = {"message": found}
    if isinstance(found, dict):
        code, message = found.get("code"), found.get("message")
        if isinstance(code, str) and code:
            error["code"] = code
        if isinstance(message, str):
            error["message"] = message
    return error


WebhookVerificationErrorCode = Literal[
    "MISSING_SIGNATURE",
    "MISSING_TIMESTAMP",
    "INVALID_SIGNATURE_FORMAT",
    "INVALID_TIMESTAMP",
    "TIMESTAMP_OUT_OF_RANGE",
    "SIGNATURE_MISMATCH",
    "INVALID_BODY",
    "INVALID_JSON",
]


class WebhookVerificationError(Exception):
    """A request meeng did not sign, signed too long ago, or that is not an event."""

    def __init__(self, code: WebhookVerificationErrorCode, message: str) -> None:
        super().__init__(message)
        self.code = code
        self.message = message

    def __str__(self) -> str:
        return f"{self.code}: {self.message}"
