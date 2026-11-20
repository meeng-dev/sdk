"""Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client at
another app, the address from MEENG_BASE_URL.

    MEENG_API_KEY=mg_xxx PYTHONPATH=src python3 examples/send_email.py

In your own project the import is the same: ``from meeng import Meeng``.
"""

from meeng import Meeng, MeengHTTPError

meeng = Meeng()

try:
    sent, _ = meeng.emails.send(
        {
            "from": "Acme <contato@acme.com.br>",
            "to": "ana@example.com",
            "subject": "Bem-vinda à Acme",
            "html": "<p>Olá, Ana! Sua conta está pronta.</p>",
            "text": "Olá, Ana! Sua conta está pronta.",
        },
        # A key makes the send safe to repeat: the same key with the same body
        # answers the first result again, and nothing is sent twice.
        options={"idempotency_key": "boas-vindas-ana-2026-11-20"},
    )
except MeengHTTPError as error:
    raise SystemExit(f"{error.status_code} {error.error['code']}: {error.error['message']}")

print("e-mail na fila:", sent["emailId"] if sent else None)
