from __future__ import annotations

from typing import Any, NotRequired, Protocol, TypedDict
from urllib.parse import quote

from ._errors import ErrorResponse


class Requester(Protocol):
    """What a service calls the API through."""

    def request(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, Any] | None = None,
        body: Any = None,
        idempotency_key: str | None = None,
    ) -> tuple[Any, ErrorResponse | None]: ...


def segment(value: object) -> str:
    """A path parameter, escaped."""
    return quote(str(value), safe="")


class RequestOptions(TypedDict):
    """Options of a call that takes an idempotency key."""

    idempotency_key: NotRequired[str | None]
    """Makes a send safe to retry: the same key with the same body answers the first result again; with another body, a 409 NOT_UNIQUE. Up to 256 characters, kept for 24 hours."""
