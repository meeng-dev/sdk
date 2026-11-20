import Foundation

#if canImport(FoundationNetworking)
    import FoundationNetworking
#endif

/// Sends one request and answers its body and response, whatever the status.
/// The SDK sends through a `URLSession` that refuses redirects; a test, or a program with its
/// own HTTP stack, gives another one to `MeengClient`.
public typealias MeengTransport = @Sendable (URLRequest) async throws -> (Data, HTTPURLResponse)

/// What a service calls the API through.
struct Requester: Sendable {
    let apiKey: String
    let baseURL: String
    let timeout: TimeInterval
    let transport: MeengTransport
    /// How many times a call that may be repeated is repeated after the first attempt.
    var maxRetries = 2
    /// The wait before the first repetition, in seconds.
    var retryDelay: TimeInterval = 0.5
    /// Why the client cannot call (no key, an insecure address); every call throws it before sending anything.
    var setupError: MeengError?

    /// Calls the API and reads its answer; a refusal throws `MeengError`.
    func request<T: Decodable>(
        _ method: String,
        _ path: String,
        query: [(String, String?)] = [],
        body: (any Encodable & Sendable)? = nil,
        idempotencyKey: String? = nil
    ) async throws -> T {
        let data = try await send(method, path, query: query, body: body, idempotencyKey: idempotencyKey)
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func send(
        _ method: String,
        _ path: String,
        query: [(String, String?)],
        body: (any Encodable & Sendable)?,
        idempotencyKey: String?
    ) async throws -> Data {
        if let setupError {
            throw setupError
        }
        var address = baseURL + MeengClient.apiPrefix + path
        let pairs = query.compactMap { name, value -> String? in
            guard let value, !value.isEmpty else { return nil }
            return "\(escape(name))=\(escape(value))"
        }
        if !pairs.isEmpty {
            address += "?" + pairs.joined(separator: "&")
        }
        guard let url = URL(string: address) else {
            throw URLError(.badURL)
        }
        var request = URLRequest(url: url, timeoutInterval: timeout)
        request.httpMethod = method
        request.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        request.setValue("meeng-swift/\(MeengClient.version)", forHTTPHeaderField: "User-Agent")
        if let body {
            let encoder = JSONEncoder()
            encoder.outputFormatting = [.withoutEscapingSlashes]
            request.httpBody = try encoder.encode(body)
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        if let idempotencyKey, !idempotencyKey.isEmpty {
            request.setValue(idempotencyKey, forHTTPHeaderField: "Idempotency-Key")
        }
        let mayRepeat = ["GET", "HEAD", "PUT", "DELETE"].contains(method.uppercased()) || !(idempotencyKey ?? "").isEmpty
        var attempt = 0
        while true {
            let data: Data
            let response: HTTPURLResponse
            do {
                (data, response) = try await transport(request)
            } catch let failure as URLError where mayRepeat && attempt < maxRetries && failure.code != .cancelled {
                // The connection failed or timed out: the call may or may not have arrived.
                try await pause(attempt: attempt, retryAfter: nil)
                attempt += 1
                continue
            }
            let status = response.statusCode
            let again = status == 429 || (mayRepeat && [500, 502, 503, 504].contains(status))
            if again && attempt < maxRetries {
                try await pause(attempt: attempt, retryAfter: response.value(forHTTPHeaderField: "Retry-After"))
                attempt += 1
                continue
            }
            if status >= 300 {
                throw MeengError(response: response, body: data)
            }
            return data
        }
    }

    /// Sleeps before repetition number `attempt` + 1: the API's Retry-After, or the
    /// base delay doubled each time and spread over its upper half.
    private func pause(attempt: Int, retryAfter: String?) async throws {
        var seconds = retryAfter.flatMap { Double($0.trimmingCharacters(in: .whitespaces)) } ?? 0
        if seconds <= 0 {
            let delay = min(retryDelay * Double(1 << min(attempt, 10)), MeengClient.maxRetryDelay)
            seconds = delay / 2 + Double.random(in: 0...(delay / 2))
        }
        try await Task.sleep(nanoseconds: UInt64(min(seconds, MeengClient.maxRetryDelay) * 1_000_000_000))
    }

    /// A session that refuses redirects: the API never redirects, and a
    /// redirect would carry the key along. The 3xx is the answer.
    private static let session = URLSession(
        configuration: .default, delegate: RefuseRedirects(), delegateQueue: nil)

    /// Sends through a `URLSession` that does not follow redirects.
    static let urlSession: MeengTransport = { request in
        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse else {
            throw URLError(.badServerResponse)
        }
        return (data, http)
    }
}

extension MeengError {
    /// The refusal of a base URL that is not https and not a loopback address.
    static func insecureBaseURL(_ url: URL) -> MeengError? {
        let scheme = url.scheme?.lowercased()
        let host = (url.host ?? "").trimmingCharacters(in: CharacterSet(charactersIn: "[]")).lowercased()
        let octets = host.split(separator: ".", omittingEmptySubsequences: false).map { Int($0) }
        let loopback =
            host == "localhost" || host == "::1"
            || (octets.count == 4 && octets[0] == 127 && octets.allSatisfy { ($0 ?? 256) <= 255 })
        if scheme == "https" || (scheme == "http" && loopback) {
            return nil
        }
        return MeengError(status: 0, code: "insecure_base_url", message: "meeng: the base URL must be https (http is accepted only for localhost, 127.0.0.0/8 and ::1), so the API key is never sent in clear text")
    }
}

/// A body with nothing in it, which a POST or PUT with no body of its own sends
/// as `{}`.
struct EmptyBody: Encodable, Sendable {}

private let unreserved = CharacterSet(charactersIn: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~")

/// A path parameter or a query value, escaped: everything but letters, digits
/// and `-._~`.
func escape(_ value: String) -> String {
    value.addingPercentEncoding(withAllowedCharacters: unreserved) ?? value
}

private final class RefuseRedirects: NSObject, URLSessionTaskDelegate, @unchecked Sendable {
    func urlSession(
        _ session: URLSession, task: URLSessionTask,
        willPerformHTTPRedirection response: HTTPURLResponse, newRequest request: URLRequest,
        completionHandler: @escaping (URLRequest?) -> Void
    ) {
        completionHandler(nil)
    }
}
