import Meeng

/// Sends one e-mail.
func sendEmail() async throws {
    let meeng = MeengClient()
    let sent = try await meeng.emails.send(
        EmailsSendParams(
            to: ["ana@example.com"],
            from: "Acme <contato@acme.com.br>",
            subject: "Bem-vinda à Acme",
            text: "Olá, Ana! Sua conta está pronta.",
            html: "<p>Olá, Ana! Sua conta está pronta.</p>"),
        // A key makes the send safe to repeat: the same key with the same body
        // answers the first result again, and nothing is sent twice.
        idempotencyKey: "boas-vindas-ana-2026-11-20")
    print("e-mail na fila: \(sent.emailID)")
}
