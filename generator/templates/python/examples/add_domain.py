"""Adds a sending domain, prints the DNS records to publish and asks for them to
be checked.

    MEENG_API_KEY=mg_xxx PYTHONPATH=src python3 examples/add_domain.py
"""

from meeng import Meeng, MeengHTTPError

meeng = Meeng()

try:
    domain, _ = meeng.domains.create({"name": "acme.com.br", "region": "us-east-1"})
    assert domain is not None
    print("publique estes registros no DNS de", domain["name"])
    for record in domain["dnsRecords"]:
        print(f"  {record['type']:<5} {record['name']:<40} {record['value']}")

    meeng.domains.verify(domain["id"])
    checked, _ = meeng.domains.get(domain["id"])
    print("situação:", checked["status"] if checked else None)
except MeengHTTPError as error:
    raise SystemExit(f"{error.status_code} {error.error['code']}: {error.error['message']}")
