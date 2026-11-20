"""Sends up to 100 e-mails in one request.

    MEENG_API_KEY=mg_xxx PYTHONPATH=src python3 examples/send_batch.py
"""

from meeng import Meeng, MeengHTTPError

meeng = Meeng()
people = [("Ana", "ana@example.com"), ("Bruno", "bruno@example.com"), ("Carla", "carla@example.com")]

try:
    sent, _ = meeng.emails.batch(
        [
            {
                "from": "Acme <contato@acme.com.br>",
                "to": email,
                "subject": "Seu extrato de novembro",
                "html": f"<p>Olá, {name}! Seu extrato está disponível.</p>",
            }
            for name, email in people
        ],
        options={"idempotency_key": "extrato-2026-11"},
    )
except MeengHTTPError as error:
    raise SystemExit(f"{error.status_code} {error.error['code']}: {error.error['message']}")

for number, email in enumerate(sent["data"] if sent else [], start=1):
    print(f"{number}: {email['emailId']}")
