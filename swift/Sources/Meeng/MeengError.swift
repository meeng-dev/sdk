import Foundation

#if canImport(FoundationNetworking)
    import FoundationNetworking
#endif

/// A request the API refused.
///
/// `code` is the API's, such as `NOT_UNIQUE`, `FORBIDDEN` or `NOT_FOUND`; it is
/// `INTERNAL_SERVER_ERROR` when the answer carried none. `retryAfter` is how
/// many seconds to wait on a 429.
public struct MeengError: Error, Sendable, Hashable, CustomStringConvertible, LocalizedError {
    /// The HTTP status; 0 for an error that never reached the API.
    public let status: Int
    /// The API's code for the refusal.
    public let code: String
    /// What went wrong, for a person to read.
    public let message: String
    /// How many seconds to wait before trying again.
    public let retryAfter: TimeInterval?

    public init(status: Int, code: String, message: String = "", retryAfter: TimeInterval? = nil) {
        self.status = status
        self.code = code
        self.message = message
        self.retryAfter = retryAfter
    }

    /// The error a refused response carries: `{"error": {"code", "message"}}`,
    /// `{"error": "text"}` or the error itself. A refusal without a JSON body is
    /// still a refusal.
    init(response: HTTPURLResponse, body: Data) {
        var code: String?
        var message: String?
        if let json = try? JSONSerialization.jsonObject(with: body) as? [String: Any] {
            let found = json["error"] ?? json
            if let text = found as? String {
                message = text
            } else if let object = found as? [String: Any] {
                code = object["code"] as? String
                message = object["message"] as? String
            }
        }
        let retryAfter = response.value(forHTTPHeaderField: "Retry-After")
            .flatMap { Int($0.trimmingCharacters(in: .whitespaces)) }
            .map(TimeInterval.init)
        self.init(
            status: response.statusCode,
            code: code.flatMap { $0.isEmpty ? nil : $0 } ?? "INTERNAL_SERVER_ERROR",
            message: message.flatMap { $0.isEmpty ? nil : $0 }
                ?? HTTPURLResponse.localizedString(forStatusCode: response.statusCode).capitalized,
            retryAfter: retryAfter
        )
    }

    public var description: String {
        message.isEmpty ? "meeng: \(status) \(code)" : "meeng: \(status) \(code): \(message)"
    }

    public var errorDescription: String? { description }
}
