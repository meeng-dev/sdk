import Foundation
import Meeng
import XCTest

#if canImport(CryptoKit)
    import CryptoKit
#endif
#if canImport(Network)
    import Network
#endif

/// The requests a fake transport was sent.
final class Recorder: @unchecked Sendable {
    private let lock = NSLock()
    private var requests: [URLRequest] = []

    func append(_ request: URLRequest) {
        lock.withLock { requests.append(request) }
    }

    var seen: [URLRequest] {
        lock.withLock { requests }
    }
}

typealias Answer = @Sendable (URLRequest) -> (status: Int, body: Any?)

func client(_ answer: @escaping Answer) -> (MeengClient, Recorder) {
    let recorder = Recorder()
    let transport: MeengTransport = { request in
        recorder.append(request)
        let (status, body) = answer(request)
        let data: Data
        if let text = body as? String {
            data = Data(text.utf8)
        } else {
            data = try body.map { try JSONSerialization.data(withJSONObject: $0) } ?? Data()
        }
        let response = HTTPURLResponse(url: request.url!, statusCode: status, httpVersion: "HTTP/1.1", headerFields: ["Retry-After": "7"])!
        return (data, response)
    }
    let meeng = MeengClient(apiKey: "mg_test_key", baseURL: URL(string: "https://meeng.test/")!, transport: transport)
    return (meeng, recorder)
}

func bodyJSON(_ request: URLRequest) -> Any? {
    request.httpBody.flatMap { try? JSONSerialization.jsonObject(with: $0) }
}

final class MeengTests: XCTestCase {
    func testSendingAnEmailCarriesTheKeyTheBodyAndTheIdempotencyKey() async throws {
        let (meeng, recorder) = client { _ in (200, ["emailId": "em_1"]) }
        let sent = try await meeng.emails.send(
            EmailsSendParams(to: ["a@x.com", "b@x.com"], from: "hello@acme.com", subject: "Hi", html: "<p>Hi</p>"),
            idempotencyKey: "order-1")
        XCTAssertEqual(sent.emailID, "em_1")
        let seen = recorder.seen[0]
        XCTAssertEqual(seen.httpMethod, "POST")
        XCTAssertEqual(seen.url?.absoluteString, "https://meeng.test/api/v1/emails")
        XCTAssertEqual(seen.value(forHTTPHeaderField: "Authorization"), "Bearer mg_test_key")
        XCTAssertEqual(seen.value(forHTTPHeaderField: "Idempotency-Key"), "order-1")
        XCTAssertEqual(seen.value(forHTTPHeaderField: "Content-Type"), "application/json")
        XCTAssertEqual(seen.value(forHTTPHeaderField: "User-Agent"), "meeng-swift/\(MeengClient.version)")
        let body = try XCTUnwrap(bodyJSON(seen) as? [String: Any])
        XCTAssertEqual(body["from"] as? String, "hello@acme.com")
        XCTAssertEqual(body["to"] as? [String], ["a@x.com", "b@x.com"])
        XCTAssertEqual(body["html"] as? String, "<p>Hi</p>")
        XCTAssertNil(body["cc"])
    }

    func testAWriteWithNoBodyOfItsOwnSendsAnEmptyObject() async throws {
        let (meeng, recorder) = client { _ in (200, ["emailId": "em_1"]) }
        _ = try await meeng.emails.cancel("em_1")
        let seen = recorder.seen[0]
        XCTAssertEqual(seen.httpMethod, "POST")
        XCTAssertEqual(seen.url?.path, "/api/v1/emails/em_1/cancel")
        XCTAssertEqual(seen.httpBody.map { String(decoding: $0, as: UTF8.self) }, "{}")
    }

    func testASendWithoutAnIdempotencyKeySendsNone() async throws {
        let (meeng, recorder) = client { _ in (200, ["emailId": "em_1"]) }
        _ = try await meeng.emails.send(EmailsSendParams(to: ["a@x.com"], from: "hello@acme.com", text: "Hi"))
        XCTAssertNil(recorder.seen[0].value(forHTTPHeaderField: "Idempotency-Key"))
    }

    func testABatchAnswersTheListOfIDs() async throws {
        let (meeng, recorder) = client { _ in (200, ["data": [["emailId": "em_1"], ["emailId": "em_2"]]]) }
        let got = try await meeng.emails.batch([
            EmailsBatchParams(to: ["a@x.com"], from: "hello@acme.com", subject: "One", text: "1"),
            EmailsBatchParams(to: ["b@x.com"], from: "hello@acme.com", subject: "Two", text: "2"),
        ])
        XCTAssertEqual(got.data[1].emailID, "em_2")
        XCTAssertEqual(recorder.seen[0].url?.path, "/api/v1/emails/batch")
        XCTAssertEqual((bodyJSON(recorder.seen[0]) as? [Any])?.count, 2)
    }

    func testListingEmailsSendsTheFiltersAListOnceForEachValue() async throws {
        let item: [String: Any] = [
            "id": "em_1", "to": "a@x.com", "from": "h@acme.com", "subject": "Hi", "html": NSNull(), "text": "Hi",
            "createdAt": "x", "updatedAt": "x", "latestStatus": "DELIVERED", "scheduledAt": NSNull(), "domainId": 3,
        ]
        nonisolated(unsafe) let page: [String: Any] = ["data": [item], "count": 1]
        let (meeng, recorder) = client { _ in (200, page) }
        let got = try await meeng.emails.list(page: 2, limit: 10, startDate: "2026-11-01T00:00:00Z", domainID: ["1", "2"])
        XCTAssertEqual(got.count, 1)
        XCTAssertEqual(Array(got.data[0].to), ["a@x.com"])
        XCTAssertEqual(got.data[0].latestStatus, EmailsListItemLatestStatus.delivered)
        XCTAssertEqual(
            recorder.seen[0].url?.absoluteString,
            "https://meeng.test/api/v1/emails?page=2&limit=10&startDate=2026-11-01T00%3A00%3A00Z&domainId=1&domainId=2")
    }

    func testAPathParameterIsEscaped() async throws {
        let (meeng, recorder) = client { _ in
            (
                200,
                [
                    "id": "a/b c", "teamId": 1, "to": ["a@x.com"], "from": "h@acme.com", "subject": "Hi", "html": NSNull(),
                    "text": NSNull(), "createdAt": "x", "updatedAt": "x",
                    "emailEvents": [["emailId": "a/b c", "status": "SENT", "createdAt": "x", "data": ["ip": "1.2.3.4"]]],
                ]
            )
        }
        let email = try await meeng.emails.get("a/b c")
        XCTAssertEqual(recorder.seen[0].url?.absoluteString, "https://meeng.test/api/v1/emails/a%2Fb%20c")
        XCTAssertEqual(email.emailEvents[0].status, EmailEventStatus.sent)
    }

    func testAListTheAPIAnswersWithoutAnObjectAroundItIsAList() async throws {
        let (meeng, _) = client { _ in
            (
                200,
                [
                    [
                        "id": "cb_1", "name": "News", "teamId": 1, "properties": ["plan": "string"], "variables": ["plan"],
                        "emoji": "x", "createdAt": "x", "updatedAt": "x", "_count": ["contacts": 4],
                    ]
                ]
            )
        }
        let books = try await meeng.contactBooks.list()
        XCTAssertEqual(books[0].name, "News")
        XCTAssertEqual(books[0].count?.contacts, 4)
    }

    func testADeleteCanCarryABody() async throws {
        let (meeng, recorder) = client { _ in (200, ["success": true, "count": 2]) }
        let got = try await meeng.contacts.bulkDelete("cb_1", ContactsBulkDeleteParams(contactIds: ["c1", "c2"]))
        XCTAssertEqual(got.count, 2)
        let seen = recorder.seen[0]
        XCTAssertEqual(seen.httpMethod, "DELETE")
        XCTAssertEqual(seen.url?.path, "/api/v1/contactBooks/cb_1/contacts/bulk")
        XCTAssertEqual((bodyJSON(seen) as? [String: Any])?["contactIds"] as? [String], ["c1", "c2"])
    }

    func testUpsertTakesTheContactsIDInThePath() async throws {
        let (meeng, recorder) = client { _ in (200, ["contactId": "c_1"]) }
        let got = try await meeng.contacts.upsert("cb_1", "c_1", ContactsUpsertParams(email: "a@x.com", firstName: "Ana"))
        XCTAssertEqual(got.contactID, "c_1")
        XCTAssertEqual(recorder.seen[0].httpMethod, "PUT")
        XCTAssertEqual(recorder.seen[0].url?.path, "/api/v1/contactBooks/cb_1/contacts/c_1")
    }

    func testARefusalThrowsInEveryShapeTheAPIWritesIt() async throws {
        let cases: [(Int, Any, String, String)] = [
            (409, ["error": ["code": "NOT_UNIQUE", "message": "Idempotency-Key already used"]], "NOT_UNIQUE", "Idempotency-Key already used"),
            (403, ["error": "Forbidden"], "INTERNAL_SERVER_ERROR", "Forbidden"),
            (400, ["code": "BAD_REQUEST", "message": "bad"], "BAD_REQUEST", "bad"),
            (
                400,
                [
                    "success": false,
                    "error": ["issues": [["code": "invalid_type", "path": ["from"], "message": "Required"]], "name": "ZodError"],
                ], "ZodError", "from: Required"
            ),
            (502, "<html>", "INTERNAL_SERVER_ERROR", "Bad Gateway"),
        ]
        for (status, body, code, message) in cases {
            nonisolated(unsafe) let payload = body
            let (meeng, _) = client { _ in (status, payload) }
            do {
                _ = try await meeng.emails.cancel("em_1")
                XCTFail("a \(status) did not throw")
            } catch let error as MeengError {
                XCTAssertEqual(error.status, status)
                XCTAssertEqual(error.code, code)
                XCTAssertEqual(error.retryAfter, 7)
                XCTAssertEqual(error.description, "meeng: \(status) \(code): \(message)")
            }
        }
    }

    func testAnEmailsRecipientsMayComeAsOneStringOrAList() async throws {
        let (meeng, _) = client { _ in
            (
                200,
                [
                    "id": "em_1", "teamId": 1, "to": "a@x.com", "from": "h@acme.com", "subject": "Hi", "html": NSNull(),
                    "text": NSNull(), "createdAt": "x", "updatedAt": "x", "emailEvents": [Any](),
                ]
            )
        }
        let email = try await meeng.emails.get("em_1")
        XCTAssertEqual(Array(email.to), ["a@x.com"])
    }

    func testTheAddressMustBeHTTPSExceptToLocalhost() async throws {
        for base in ["http://app.meeng.test", "ftp://app.meeng.test", "http://localhost.evil.test"] {
            let meeng = MeengClient(apiKey: "k", baseURL: URL(string: base)!, transport: { _ in XCTFail("sent"); throw URLError(.badURL) })
            do {
                _ = try await meeng.emails.cancel("em_1")
                XCTFail("\(base) was accepted")
            } catch let error as MeengError {
                XCTAssertEqual(error.code, "insecure_base_url")
            }
        }
        XCTAssertEqual(MeengClient(apiKey: "k", baseURL: URL(string: "https://app.meeng.test/")!).url, "https://app.meeng.test/api")
        XCTAssertEqual(MeengClient(apiKey: "k", baseURL: URL(string: "http://127.0.0.1:3000")!).url, "http://127.0.0.1:3000/api")
    }

    func testAMissingKeyThrowsBeforeSendingAnything() async throws {
        guard ProcessInfo.processInfo.environment["MEENG_API_KEY"] == nil else { return }
        let meeng = MeengClient(baseURL: URL(string: "https://meeng.test")!, transport: { _ in XCTFail("sent"); throw URLError(.badURL) })
        do {
            _ = try await meeng.emails.cancel("em_1")
            XCTFail("a missing key was accepted")
        } catch let error as MeengError {
            XCTAssertEqual(error.code, "missing_api_key")
        }
    }

    #if canImport(Network)
        func testTheDefaultTransportNeverFollowsARedirect() async throws {
            let listener = try NWListener(using: .tcp, on: .any)
            listener.newConnectionHandler = { connection in
                connection.start(queue: .global())
                connection.receive(minimumIncompleteLength: 1, maximumLength: 65536) { _, _, _, _ in
                    let answer = "HTTP/1.1 302 Found\r\nLocation: https://elsewhere.test/\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"
                    connection.send(content: Data(answer.utf8), completion: .contentProcessed { _ in connection.cancel() })
                }
            }
            let ready = expectation(description: "listening")
            listener.stateUpdateHandler = { state in
                if case .ready = state { ready.fulfill() }
            }
            listener.start(queue: .global())
            await fulfillment(of: [ready], timeout: 5)
            defer { listener.cancel() }
            let port = try XCTUnwrap(listener.port?.rawValue)
            let meeng = MeengClient(apiKey: "k", baseURL: URL(string: "http://127.0.0.1:\(port)")!)
            do {
                _ = try await meeng.emails.cancel("em_1")
                XCTFail("a redirect was followed or hidden")
            } catch let error as MeengError {
                XCTAssertEqual(error.status, 302)
            }
        }
    #endif

    #if canImport(CryptoKit)
        static let secret = "whsec_test"
        static let body = Data(
            #"{"id":"evt_1","type":"email.delivered","version":"2026-01-18","createdAt":"2026-11-20T12:00:00Z","teamId":7,"attempt":2,"data":{"id":"em_1","status":"DELIVERED","from":"h@acme.com","to":["a@x.com"],"occurredAt":"2026-11-20T12:00:00Z"}}"#
                .utf8)

        static func sign(_ timestamp: String, _ body: Data = body, _ secret: String = secret) -> String {
            var signed = Data("\(timestamp).".utf8)
            signed.append(body)
            let mac = HMAC<SHA256>.authenticationCode(for: signed, using: SymmetricKey(data: Data(secret.utf8)))
            return "v1=" + mac.map { String(format: "%02x", $0) }.joined()
        }

        func testASignedWebhookIsVerifiedAndReadAsATypedEvent() throws {
            let timestamp = String(Int(Date().timeIntervalSince1970 * 1000))
            let webhooks = MeengClient(apiKey: "k").webhooks(secret: Self.secret)
            let headers = ["X-Meeng-Signature": Self.sign(timestamp), "X-Meeng-Timestamp": timestamp]
            XCTAssertTrue(webhooks.verify(Self.body, headers: headers))
            guard case let .email(event) = try webhooks.constructEvent(Self.body, headers: headers) else {
                return XCTFail("not an email event")
            }
            XCTAssertEqual(event.type, "email.delivered")
            XCTAssertEqual(event.version, "2026-01-18")
            XCTAssertEqual(event.teamID, 7)
            XCTAssertEqual(event.attempt, 2)
            XCTAssertEqual(event.data.to, ["a@x.com"])
            XCTAssertEqual(event.data.status, EmailBasePayloadStatus.delivered)
        }

        func testTheUseSendHeadersAreAcceptedAsWellInAnyCase() {
            let timestamp = String(Int(Date().timeIntervalSince1970 * 1000))
            let headers = ["x-usesend-signature": Self.sign(timestamp), "X-UseSend-Timestamp": timestamp]
            XCTAssertTrue(Webhooks(secret: Self.secret).verify(Self.body, headers: headers))
        }

        func testAWebhookIsRefusedWithTheCodeThatSaysWhy() {
            let now = Date(timeIntervalSince1970: 1_800_000_000)
            let timestamp = "1800000000000"
            let old = String(1_800_000_000_000 - 6 * 60 * 1000)
            let webhooks = Webhooks(secret: Self.secret)
            let good = ["X-Meeng-Signature": Self.sign(timestamp), "X-Meeng-Timestamp": timestamp]
            let stale = ["X-Meeng-Signature": Self.sign(old), "X-Meeng-Timestamp": old]
            func refused(_ headers: [String: String], body: Data = Self.body, secret: String? = nil, tolerance: TimeInterval? = nil) -> String {
                do {
                    _ = try webhooks.constructEvent(body, headers: headers, secret: secret, tolerance: tolerance, now: now)
                    return "ACCEPTED"
                } catch let error as WebhookVerificationError {
                    return error.code
                } catch {
                    return "OTHER"
                }
            }
            XCTAssertEqual(refused(good), "ACCEPTED")
            XCTAssertEqual(refused(["X-Meeng-Timestamp": timestamp]), "MISSING_SIGNATURE")
            XCTAssertEqual(refused(["X-Meeng-Signature": Self.sign(timestamp)]), "MISSING_TIMESTAMP")
            XCTAssertEqual(refused(["X-Meeng-Signature": String(Self.sign(timestamp).dropFirst(3)), "X-Meeng-Timestamp": timestamp]), "INVALID_SIGNATURE_FORMAT")
            XCTAssertEqual(refused(["X-Meeng-Signature": Self.sign(timestamp), "X-Meeng-Timestamp": "yesterday"]), "INVALID_TIMESTAMP")
            XCTAssertEqual(refused(stale), "TIMESTAMP_OUT_OF_RANGE")
            XCTAssertEqual(refused(stale, tolerance: -1), "ACCEPTED")
            XCTAssertEqual(refused(stale, tolerance: 600), "ACCEPTED")
            XCTAssertEqual(refused(good, secret: "another"), "SIGNATURE_MISMATCH")
            XCTAssertEqual(refused(good, body: Data(String(decoding: Self.body, as: UTF8.self).replacingOccurrences(of: "em_1", with: "em_2").utf8)), "SIGNATURE_MISMATCH")
            let notJSON = Data("not json".utf8)
            XCTAssertEqual(refused(["X-Meeng-Signature": Self.sign(timestamp, notJSON), "X-Meeng-Timestamp": timestamp], body: notJSON), "INVALID_JSON")
        }
    #endif
}
