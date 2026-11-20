// Adds a sending domain, prints the DNS records to publish and asks for them to
// be checked.
//
//   MEENG_API_KEY=mg_xxx node examples/add-domain.ts
import { Meeng } from "../src/index.ts";

const meeng = new Meeng();

const created = await meeng.domains.create({ name: "acme.com.br", region: "us-east-1" });
if (created.error) {
  console.error(`${created.error.status} ${created.error.code}: ${created.error.message}`);
  process.exit(1);
}
const domain = created.data;
console.log(`publique estes registros no DNS de ${domain.name}`);
for (const record of domain.dnsRecords) {
  console.log(`  ${record.type.padEnd(5)} ${record.name.padEnd(40)} ${record.value}`);
}

await meeng.domains.verify(domain.id);
const checked = await meeng.domains.get(domain.id);
console.log("situação:", checked.data?.status);
