// Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client at
// another app, the address from MEENG_BASE_URL.
//
//   MEENG_API_KEY=mg_xxx node examples/send-email.ts
//
// In your own project the import is `import { Meeng } from "@meeng-dev/sdk"`.
import { Meeng } from "../src/index.ts";

const meeng = new Meeng();

const { data, error } = await meeng.emails.send(
  {
    from: "Acme <contato@acme.com.br>",
    to: "ana@example.com",
    subject: "Bem-vinda à Acme",
    html: "<p>Olá, Ana! Sua conta está pronta.</p>",
    text: "Olá, Ana! Sua conta está pronta.",
  },
  // A key makes the send safe to repeat: the same key with the same body answers
  // the first result again, and nothing is sent twice.
  { idempotencyKey: "boas-vindas-ana-2026-11-20" },
);

if (error) {
  console.error(`${error.status} ${error.code}: ${error.message}`);
  process.exit(1);
}
console.log("e-mail na fila:", data.emailId);
