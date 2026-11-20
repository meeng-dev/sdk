// Checks a webhook the way an endpoint does: the signature and the time are
// verified against the raw body, and then the event is read as a typed value.
// The request here is made in the program; a real endpoint reads the body and the
// headers of the request it receives, and the secret from its settings.
//
//   node examples/verify-webhook.ts
import { createHmac } from "node:crypto";

import { Meeng } from "../src/index.ts";

const secret = "whsec_exemplo";
const body = JSON.stringify({
  id: "call_1",
  type: "email.delivered",
  version: "2026-01-18",
  createdAt: "2026-11-20T12:00:00Z",
  teamId: 7,
  attempt: 1,
  data: { id: "em_1", status: "DELIVERED", from: "contato@acme.com.br", to: ["ana@example.com"], occurredAt: "2026-11-20T12:00:00Z" },
});
const timestamp = String(Date.now());
const headers = {
  "x-meeng-signature": `v1=${createHmac("sha256", secret).update(`${timestamp}.${body}`).digest("hex")}`,
  "x-meeng-timestamp": timestamp,
};

const webhooks = new Meeng("mg_exemplo").webhooks(secret);
const event = webhooks.constructEvent(body, { headers });
if (event.type === "email.delivered") {
  console.log("e-mail", event.data.id, "entregue a", event.data.to.join(", "));
} else {
  console.log("evento", event.type);
}
