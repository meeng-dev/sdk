import Meeng

/// Sends up to 100 e-mails in one request.
func sendBatch() async throws {
    let meeng = MeengClient()
    let people = [("Ana", "ana@example.com"), ("Bruno", "bruno@example.com"), ("Carla", "carla@example.com")]
    let batch = people.map { name, email in
        EmailsBatchParams(
            to: [email],
            from: "Acme <contato@acme.com.br>",
            subject: "Seu extrato de novembro",
            html: "<p>Olá, \(name)! Seu extrato está disponível.</p>")
    }
    let sent = try await meeng.emails.batch(batch, idempotencyKey: "extrato-2026-11")
    for (index, email) in sent.data.enumerated() {
        print("\(index + 1): \(email.emailID)")
    }
}
