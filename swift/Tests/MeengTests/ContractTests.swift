import Foundation
import Meeng
import XCTest

/// The contract test calls every method of the SDK against a fake of the API that is
/// built from the API's contract and refuses what the contract does not allow.
/// check.sh starts the fake and sets MEENG_FAKE_URL.
final class ContractTests: XCTestCase {
    private func contractClient() throws -> MeengClient {
        guard let base = ProcessInfo.processInfo.environment["MEENG_FAKE_URL"], let url = URL(string: base) else {
            throw XCTSkip("MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test")
        }
        return MeengClient(apiKey: "mg_contract_test", baseURL: url, retryDelay: 0.001)
    }

    func testEveryMethodMakesTheCallTheContractDescribes() async throws {
        let meeng = try contractClient()
        var failures: [String] = []
        do { _ = try await meeng.emails.send(EmailsSendParams(to: ["hello@example.com"], from: "hello@example.com"), idempotencyKey: "contract-emails-send") } catch { failures.append("emails.send: \(error)") }
        do { _ = try await meeng.emails.batch([EmailsBatchParams(to: ["hello@example.com"], from: "hello@example.com")], idempotencyKey: "contract-emails-batch") } catch { failures.append("emails.batch: \(error)") }
        do { _ = try await meeng.emails.get("id_emailid") } catch { failures.append("emails.get: \(error)") }
        do { _ = try await meeng.emails.list(page: 2, limit: 10, startDate: "2026-11-01T00:00:00Z", endDate: "2026-11-30T00:00:00Z", domainID: ["1", "2"]) } catch { failures.append("emails.list: \(error)") }
        do { _ = try await meeng.emails.update("id_emailid", EmailsUpdateParams(scheduledAt: "2026-12-01T09:00:00Z")) } catch { failures.append("emails.update: \(error)") }
        do { _ = try await meeng.emails.cancel("id_emailid") } catch { failures.append("emails.cancel: \(error)") }
        do { _ = try await meeng.domains.list() } catch { failures.append("domains.list: \(error)") }
        do { _ = try await meeng.domains.create(DomainsCreateParams(name: "example.com", region: "us-east-1")) } catch { failures.append("domains.create: \(error)") }
        do { _ = try await meeng.domains.verify("id_id") } catch { failures.append("domains.verify: \(error)") }
        do { _ = try await meeng.domains.get("id_id") } catch { failures.append("domains.get: \(error)") }
        do { _ = try await meeng.domains.delete("id_id") } catch { failures.append("domains.delete: \(error)") }
        do { _ = try await meeng.contacts.create("id_contactbookid", ContactsCreateParams(email: "hello@example.com")) } catch { failures.append("contacts.create: \(error)") }
        do { _ = try await meeng.contacts.list("id_contactbookid", emails: "a@example.com", page: 2, limit: 10, ids: "c1,c2") } catch { failures.append("contacts.list: \(error)") }
        do { _ = try await meeng.contacts.get("id_contactbookid", "id_contactid") } catch { failures.append("contacts.get: \(error)") }
        do { _ = try await meeng.contacts.update("id_contactbookid", "id_contactid", ContactsUpdateParams()) } catch { failures.append("contacts.update: \(error)") }
        do { _ = try await meeng.contacts.upsert("id_contactbookid", "id_contactid", ContactsUpsertParams(email: "hello@example.com")) } catch { failures.append("contacts.upsert: \(error)") }
        do { _ = try await meeng.contacts.delete("id_contactbookid", "id_contactid") } catch { failures.append("contacts.delete: \(error)") }
        do { _ = try await meeng.contacts.bulkCreate("id_contactbookid", [ContactsBulkCreateParams(email: "hello@example.com")]) } catch { failures.append("contacts.bulkCreate: \(error)") }
        do { _ = try await meeng.contacts.bulkDelete("id_contactbookid", ContactsBulkDeleteParams(contactIds: ["c1", "c2"])) } catch { failures.append("contacts.bulkDelete: \(error)") }
        do { _ = try await meeng.contactBooks.list() } catch { failures.append("contactBooks.list: \(error)") }
        do { _ = try await meeng.contactBooks.create(ContactBooksCreateParams(name: "example.com")) } catch { failures.append("contactBooks.create: \(error)") }
        do { _ = try await meeng.contactBooks.get("id_contactbookid") } catch { failures.append("contactBooks.get: \(error)") }
        do { _ = try await meeng.contactBooks.update("id_contactbookid", ContactBooksUpdateParams()) } catch { failures.append("contactBooks.update: \(error)") }
        do { _ = try await meeng.contactBooks.delete("id_contactbookid") } catch { failures.append("contactBooks.delete: \(error)") }
        do { _ = try await meeng.campaigns.create(CampaignsCreateParams(name: "example.com", from: "hello@example.com", subject: "sample", contactBookID: "id_contactbookid")) } catch { failures.append("campaigns.create: \(error)") }
        do { _ = try await meeng.campaigns.list(page: 2, status: "DRAFT", search: "news") } catch { failures.append("campaigns.list: \(error)") }
        do { _ = try await meeng.campaigns.get("id_campaignid") } catch { failures.append("campaigns.get: \(error)") }
        do { _ = try await meeng.campaigns.delete("id_campaignid") } catch { failures.append("campaigns.delete: \(error)") }
        do { _ = try await meeng.campaigns.schedule("id_campaignid", CampaignsScheduleParams()) } catch { failures.append("campaigns.schedule: \(error)") }
        do { _ = try await meeng.campaigns.pause("id_campaignid") } catch { failures.append("campaigns.pause: \(error)") }
        do { _ = try await meeng.campaigns.resume("id_campaignid") } catch { failures.append("campaigns.resume: \(error)") }
        do { _ = try await meeng.analytics.emailTimeSeries(days: "7", domainID: "1") } catch { failures.append("analytics.emailTimeSeries: \(error)") }
        do { _ = try await meeng.analytics.reputationMetrics(domainID: "1") } catch { failures.append("analytics.reputationMetrics: \(error)") }
        XCTAssertTrue(failures.isEmpty, failures.joined(separator: "\n"))
    }

    func testARefusalIsATypedError() async throws {
        let meeng = try contractClient()
        do {
            _ = try await meeng.emails.get("fake-404")
            XCTFail("a 404 was accepted")
        } catch let refusal as MeengError {
            XCTAssertEqual(refusal.status, 404)
            XCTAssertEqual(refusal.code, "NOT_FOUND")
        }
    }

    func testATooManyRequestsAnswerIsRepeatedUntilItPasses() async throws {
        let meeng = try contractClient()
        _ = try await meeng.domains.get("flaky-429-2")
        do {
            _ = try await meeng.domains.get("fake-429")
            XCTFail("a 429 that never passes was accepted")
        } catch let refusal as MeengError {
            XCTAssertEqual(refusal.status, 429)
            XCTAssertEqual(refusal.code, "RATE_LIMITED")
        }
    }

    func testAServerFailureIsRepeatedForARead() async throws {
        let meeng = try contractClient()
        _ = try await meeng.emails.get("flaky-503-2")
    }

    func testTheSameIdempotencyKeyAnswersTheSameEmail() async throws {
        let meeng = try contractClient()
        let key = "order-\(UInt64(Date().timeIntervalSince1970 * 1_000_000))"
        let email = EmailsSendParams(to: ["a@example.com"], from: "hello@example.com")
        let first = try await meeng.emails.send(email, idempotencyKey: key)
        let again = try await meeng.emails.send(email, idempotencyKey: key)
        XCTAssertEqual(first.emailID, again.emailID)
        do {
            _ = try await meeng.emails.send(EmailsSendParams(to: ["b@example.com"], from: "hello@example.com"), idempotencyKey: key)
            XCTFail("another body under the same key was accepted")
        } catch let refusal as MeengError {
            XCTAssertEqual(refusal.status, 409)
            XCTAssertEqual(refusal.code, "NOT_UNIQUE")
        }
    }
}
