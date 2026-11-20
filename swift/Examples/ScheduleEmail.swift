import Foundation
import Meeng

/// Schedules an e-mail, moves it to another time and cancels it.
func scheduleEmail() async throws {
    let meeng = MeengClient()
    func inHours(_ hours: Double) -> String {
        ISO8601DateFormatter().string(from: Date().addingTimeInterval(hours * 3600))
    }

    let sent = try await meeng.emails.send(
        EmailsSendParams(
            to: ["ana@example.com"],
            from: "Acme <contato@acme.com.br>",
            subject: "Lembrete: sua reunião é amanhã",
            html: "<p>Nos vemos às 10h.</p>",
            scheduledAt: inHours(24)))
    print("agendado: \(sent.emailID)")

    _ = try await meeng.emails.update(sent.emailID, EmailsUpdateParams(scheduledAt: inHours(48)))
    print("remarcado")

    _ = try await meeng.emails.cancel(sent.emailID)
    print("cancelado")
}
