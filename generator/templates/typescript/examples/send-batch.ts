// Sends up to 100 e-mails in one request.
//
//   MEENG_API_KEY=mg_xxx node examples/send-batch.ts
import { Meeng } from "../src/index.ts";

const meeng = new Meeng();
const people = [
  { name: "Ana", email: "ana@example.com" },
  { name: "Bruno", email: "bruno@example.com" },
  { name: "Carla", email: "carla@example.com" },
];

const { data, error } = await meeng.emails.batch(
  people.map(({ name, email }) => ({
    from: "Acme <contato@acme.com.br>",
    to: email,
    subject: "Seu extrato de novembro",
    html: `<p>Olá, ${name}! Seu extrato está disponível.</p>`,
  })),
  { idempotencyKey: "extrato-2026-11" },
);

if (error) {
  console.error(`${error.status} ${error.code}: ${error.message}`);
  process.exit(1);
}
data.forEach((email, index) => console.log(`${index + 1}: ${email.emailId}`));
