"""Checks a webhook the way an endpoint does: the signature and the time are
verified against the raw body, and then the event is read.
The request here is made in the program; a real endpoint reads the body and the
headers of the request it receives, and the secret from its settings.

    PYTHONPATH=src python3 examples/verify_webhook.py
"""

import hashlib
import hmac
import json
import time

from meeng import Meeng, WebhookVerificationError

secret = "whsec_exemplo"
body = json.dumps(
    {
        "id": "call_1",
        "type": "email.delivered",
        "version": "2026-01-18",
        "createdAt": "2026-11-20T12:00:00Z",
        "teamId": 7,
        "attempt": 1,
        "data": {"id": "em_1", "status": "DELIVERED", "from": "contato@acme.com.br", "to": ["ana@example.com"], "occurredAt": "2026-11-20T12:00:00Z"},
    }
)
timestamp = str(int(time.time() * 1000))
signature = hmac.new(secret.encode(), f"{timestamp}.{body}".encode(), hashlib.sha256).hexdigest()
headers = {"x-meeng-signature": f"v1={signature}", "x-meeng-timestamp": timestamp}

try:
    event = Meeng("mg_exemplo").webhooks(secret).construct_event(body, headers=headers)
except WebhookVerificationError as error:
    raise SystemExit(f"{error.code}: {error}")

if event["type"] == "email.delivered":
    print("e-mail", event["data"]["id"], "entregue a", ", ".join(event["data"]["to"]))
else:
    print("evento", event["type"])
