# frozen_string_literal: true

# The contract test calls every method of the SDK against a fake of the API that is
# built from the API's contract and refuses what the contract does not allow.
# check.sh starts the fake and sets MEENG_FAKE_URL.

require "minitest/autorun"
require "meeng"

class ContractTest < Minitest::Test
  FAKE = ENV.fetch("MEENG_FAKE_URL", nil)

  def setup
    skip "MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test" if FAKE.nil? || FAKE.empty?
  end

  def meeng
    @meeng ||= Meeng::Client.new("mg_contract_test", FAKE, retry_delay: 0.001)
  end

  def test_every_method_makes_the_call_the_contract_describes
    begin
      meeng.emails.send({ "to" => ["hello@example.com"], "from" => "hello@example.com" }, idempotency_key: "contract-emails-send")
    rescue Meeng::Error => e
      flunk "emails.send: #{e.message}"
    end
    begin
      meeng.emails.batch([{ "to" => ["hello@example.com"], "from" => "hello@example.com" }], idempotency_key: "contract-emails-batch")
    rescue Meeng::Error => e
      flunk "emails.batch: #{e.message}"
    end
    begin
      meeng.emails.get("id_emailid")
    rescue Meeng::Error => e
      flunk "emails.get: #{e.message}"
    end
    begin
      meeng.emails.list(page: 2, limit: 10, start_date: "2026-11-01T00:00:00Z", end_date: "2026-11-30T00:00:00Z", domain_id: ["1", "2"])
    rescue Meeng::Error => e
      flunk "emails.list: #{e.message}"
    end
    begin
      meeng.emails.update("id_emailid", { "scheduledAt" => "2026-12-01T09:00:00Z" })
    rescue Meeng::Error => e
      flunk "emails.update: #{e.message}"
    end
    begin
      meeng.emails.cancel("id_emailid")
    rescue Meeng::Error => e
      flunk "emails.cancel: #{e.message}"
    end
    begin
      meeng.domains.list()
    rescue Meeng::Error => e
      flunk "domains.list: #{e.message}"
    end
    begin
      meeng.domains.create({ "name" => "example.com", "region" => "us-east-1" })
    rescue Meeng::Error => e
      flunk "domains.create: #{e.message}"
    end
    begin
      meeng.domains.verify("id_id")
    rescue Meeng::Error => e
      flunk "domains.verify: #{e.message}"
    end
    begin
      meeng.domains.get("id_id")
    rescue Meeng::Error => e
      flunk "domains.get: #{e.message}"
    end
    begin
      meeng.domains.delete("id_id")
    rescue Meeng::Error => e
      flunk "domains.delete: #{e.message}"
    end
    begin
      meeng.contacts.create("id_contactbookid", { "email" => "hello@example.com" })
    rescue Meeng::Error => e
      flunk "contacts.create: #{e.message}"
    end
    begin
      meeng.contacts.list("id_contactbookid", emails: "a@example.com", page: 2, limit: 10, ids: "c1,c2")
    rescue Meeng::Error => e
      flunk "contacts.list: #{e.message}"
    end
    begin
      meeng.contacts.get("id_contactbookid", "id_contactid")
    rescue Meeng::Error => e
      flunk "contacts.get: #{e.message}"
    end
    begin
      meeng.contacts.update("id_contactbookid", "id_contactid", {  })
    rescue Meeng::Error => e
      flunk "contacts.update: #{e.message}"
    end
    begin
      meeng.contacts.upsert("id_contactbookid", "id_contactid", { "email" => "hello@example.com" })
    rescue Meeng::Error => e
      flunk "contacts.upsert: #{e.message}"
    end
    begin
      meeng.contacts.delete("id_contactbookid", "id_contactid")
    rescue Meeng::Error => e
      flunk "contacts.delete: #{e.message}"
    end
    begin
      meeng.contacts.bulk_create("id_contactbookid", [{ "email" => "hello@example.com" }])
    rescue Meeng::Error => e
      flunk "contacts.bulkCreate: #{e.message}"
    end
    begin
      meeng.contacts.bulk_delete("id_contactbookid", { "contactIds" => ["c1", "c2"] })
    rescue Meeng::Error => e
      flunk "contacts.bulkDelete: #{e.message}"
    end
    begin
      meeng.contact_books.list()
    rescue Meeng::Error => e
      flunk "contactBooks.list: #{e.message}"
    end
    begin
      meeng.contact_books.create({ "name" => "example.com" })
    rescue Meeng::Error => e
      flunk "contactBooks.create: #{e.message}"
    end
    begin
      meeng.contact_books.get("id_contactbookid")
    rescue Meeng::Error => e
      flunk "contactBooks.get: #{e.message}"
    end
    begin
      meeng.contact_books.update("id_contactbookid", {  })
    rescue Meeng::Error => e
      flunk "contactBooks.update: #{e.message}"
    end
    begin
      meeng.contact_books.delete("id_contactbookid")
    rescue Meeng::Error => e
      flunk "contactBooks.delete: #{e.message}"
    end
    begin
      meeng.campaigns.create({ "name" => "example.com", "from" => "hello@example.com", "subject" => "sample", "contactBookId" => "id_contactbookid" })
    rescue Meeng::Error => e
      flunk "campaigns.create: #{e.message}"
    end
    begin
      meeng.campaigns.list(page: 2, status: "DRAFT", search: "news")
    rescue Meeng::Error => e
      flunk "campaigns.list: #{e.message}"
    end
    begin
      meeng.campaigns.get("id_campaignid")
    rescue Meeng::Error => e
      flunk "campaigns.get: #{e.message}"
    end
    begin
      meeng.campaigns.delete("id_campaignid")
    rescue Meeng::Error => e
      flunk "campaigns.delete: #{e.message}"
    end
    begin
      meeng.campaigns.schedule("id_campaignid", {  })
    rescue Meeng::Error => e
      flunk "campaigns.schedule: #{e.message}"
    end
    begin
      meeng.campaigns.pause("id_campaignid")
    rescue Meeng::Error => e
      flunk "campaigns.pause: #{e.message}"
    end
    begin
      meeng.campaigns.resume("id_campaignid")
    rescue Meeng::Error => e
      flunk "campaigns.resume: #{e.message}"
    end
    begin
      meeng.analytics.email_time_series(days: "7", domain_id: "1")
    rescue Meeng::Error => e
      flunk "analytics.emailTimeSeries: #{e.message}"
    end
    begin
      meeng.analytics.reputation_metrics(domain_id: "1")
    rescue Meeng::Error => e
      flunk "analytics.reputationMetrics: #{e.message}"
    end
  end

  def test_a_refusal_is_a_typed_error
    error = assert_raises(Meeng::Error) { meeng.emails.get("fake-404") }
    assert_equal 404, error.status
    assert_equal "NOT_FOUND", error.code
  end

  def test_a_too_many_requests_answer_is_repeated_until_it_passes
    meeng.domains.get("flaky-429-2")
    error = assert_raises(Meeng::Error) { meeng.domains.get("fake-429") }
    assert_equal 429, error.status
    assert_equal "RATE_LIMITED", error.code
  end

  def test_a_server_failure_is_repeated_for_a_read
    meeng.emails.get("flaky-503-2")
  end

  def test_the_same_idempotency_key_answers_the_same_email
    key = "order-#{Process.clock_gettime(Process::CLOCK_REALTIME, :nanosecond)}"
    email = { from: "hello@example.com", to: ["a@example.com"] }
    first = meeng.emails.send(email, idempotency_key: key)
    again = meeng.emails.send(email, idempotency_key: key)
    assert_equal first[:emailId], again[:emailId]
    error = assert_raises(Meeng::Error) { meeng.emails.send({ from: "hello@example.com", to: ["b@example.com"] }, idempotency_key: key) }
    assert_equal 409, error.status
    assert_equal "NOT_UNIQUE", error.code
  end

  def test_an_email_needs_its_required_fields
    error = assert_raises(Meeng::Error) { meeng.emails.send({ to: "a@example.com" }) }
    assert_equal 400, error.status
    assert_equal "ZodError", error.code
  end
end
