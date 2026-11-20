"""Schedules an e-mail, moves it to another time and cancels it.

    MEENG_API_KEY=mg_xxx PYTHONPATH=src python3 examples/schedule_email.py
"""

from datetime import datetime, timedelta, timezone

from meeng import Meeng, MeengHTTPError


def in_hours(hours: int) -> str:
    return (datetime.now(timezone.utc) + timedelta(hours=hours)).strftime("%Y-%m-%dT%H:%M:%SZ")


meeng = Meeng()

try:
    sent, _ = meeng.emails.send(
        {
            "from": "Acme <contato@acme.com.br>",
            "to": "ana@example.com",
            "subject": "Lembrete: sua reunião é amanhã",
            "html": "<p>Nos vemos às 10h.</p>",
            "scheduledAt": in_hours(24),
        }
    )
    assert sent is not None
    email_id = sent["emailId"]
    print("agendado:", email_id)

    meeng.emails.update(email_id, {"scheduledAt": in_hours(48)})
    print("remarcado")

    meeng.emails.cancel(email_id)
    print("cancelado")
except MeengHTTPError as error:
    raise SystemExit(f"{error.status_code} {error.error['code']}: {error.error['message']}")
