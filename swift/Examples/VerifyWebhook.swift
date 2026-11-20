import CryptoKit
import Foundation
import Meeng

/// Checks a webhook the way an endpoint does: the signature and the time are
/// verified against the raw body, and then the event is read as a typed value.
/// The request here is made in the program; a real endpoint reads the body and the
/// headers of the request it receives, and the secret from its settings.
func verifyWebhook() async throws {
    let secret = "whsec_exemplo"
    let body = #"{"id":"call_1","type":"email.delivered","version":"2026-01-18","createdAt":"2026-11-20T12:00:00Z","teamId":7,"attempt":1,"data":{"id":"em_1","status":"DELIVERED","from":"contato@acme.com.br","to":["ana@example.com"],"occurredAt":"2026-11-20T12:00:00Z"}}"#

    let timestamp = String(Int(Date().timeIntervalSince1970 * 1000))
    let mac = HMAC<SHA256>.authenticationCode(for: Data("\(timestamp).\(body)".utf8), using: SymmetricKey(data: Data(secret.utf8)))
    let signature = mac.map { String(format: "%02x", $0) }.joined()
    let headers = ["X-Meeng-Signature": "v1=\(signature)", "X-Meeng-Timestamp": timestamp]

    switch try Webhooks(secret: secret).constructEvent(Data(body.utf8), headers: headers) {
    case let .email(event):
        print("e-mail \(event.data.id) entregue a \(event.data.to.joined(separator: ", "))")
    case let other:
        print("evento \(other.type)")
    }
}
