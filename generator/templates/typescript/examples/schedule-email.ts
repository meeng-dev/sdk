// Schedules an e-mail, moves it to another time and cancels it.
//
//   MEENG_API_KEY=mg_xxx node examples/schedule-email.ts
import { Meeng } from "../src/index.ts";

const meeng = new Meeng();
const inHours = (hours: number) => new Date(Date.now() + hours * 3_600_000).toISOString();

const sent = await meeng.emails.send({
  from: "Acme <contato@acme.com.br>",
  to: "ana@example.com",
  subject: "Lembrete: sua reunião é amanhã",
  html: "<p>Nos vemos às 10h.</p>",
  scheduledAt: inHours(24),
});
if (sent.error) {
  console.error(`${sent.error.status} ${sent.error.code}: ${sent.error.message}`);
  process.exit(1);
}
const { emailId } = sent.data;
console.log("agendado:", emailId);

const moved = await meeng.emails.update(emailId, { scheduledAt: inHours(48) });
console.log("remarcado:", moved.error ? moved.error.message : "ok");

const cancelled = await meeng.emails.cancel(emailId);
console.log("cancelado:", cancelled.error ? cancelled.error.message : "ok");
