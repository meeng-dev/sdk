# frozen_string_literal: true

# meeng's Ruby SDK tests, with nothing to install: ruby -Ilib -Itest test/meeng_test.rb

require "minitest/autorun"
require "json"
require "openssl"
require "socket"
require "uri"
require "meeng"

class MeengTest < Minitest::Test
  # Records each request and answers what the test says.
  class Recorder
    attr_reader :seen

    def initialize(&answer)
      @answer = answer
      @seen = []
    end

    def call(method, url, headers, body, _timeout)
      request = { method: method, url: URI(url), headers: headers, body: body }
      @seen << request
      status, payload = @answer.call(request)
      [status, { "retry-after" => "7" }, payload.is_a?(String) ? payload : JSON.generate(payload)]
    end
  end

  def client(&answer)
    recorder = Recorder.new(&answer)
    [Meeng::Client.new("mg_test_key", "https://meeng.test/", transport: recorder, max_retries: 0), recorder]
  end

  # A client that waits a millisecond between attempts; the block gets the number
  # of the attempt and answers [status, payload] or raises.
  def retrying(max_retries: 2, &answer)
    seen = []
    transport = lambda do |method, _url, headers, _body, _timeout|
      seen << { method: method, headers: headers }
      status, payload = answer.call(seen.size)
      [status, { "retry-after" => "0" }, JSON.generate(payload)]
    end
    [Meeng::Client.new("mg_test_key", "https://meeng.test/", transport: transport, max_retries: max_retries, retry_delay: 0.001), seen]
  end

  LIMITED = [429, { "error" => { "code" => "RATE_LIMITED", "message" => "slow down" } }].freeze
  BUSY = [503, { "error" => { "code" => "INTERNAL_SERVER_ERROR", "message" => "busy" } }].freeze

  def test_a_too_many_requests_answer_is_repeated_for_any_call
    meeng, seen = retrying { |n| n < 3 ? LIMITED : [200, { "emailId" => "em_1" }] }
    assert_equal "em_1", meeng.emails.send({ from: "h@acme.com", to: "a@x.com" })[:emailId]
    assert_equal 3, seen.size
  end

  def test_a_server_failure_is_repeated_only_when_repeating_cannot_do_the_work_twice
    answer = ->(n) { n == 1 ? BUSY : [200, []] }
    meeng, seen = retrying(&answer)
    meeng.domains.list
    assert_equal 2, seen.size

    meeng, seen = retrying(&answer)
    assert_raises(Meeng::Error) { meeng.emails.send({ from: "h@acme.com", to: "a@x.com" }) }
    assert_equal 1, seen.size

    meeng, seen = retrying(&answer)
    meeng.emails.send({ from: "h@acme.com", to: "a@x.com" }, idempotency_key: "order-1")
    assert_equal 2, seen.size
    assert_equal "order-1", seen[1][:headers]["Idempotency-Key"]
  end

  def test_other_refusals_are_never_repeated_retries_can_be_turned_off_and_the_last_answer_is_the_error
    meeng, seen = retrying { [409, { "error" => { "code" => "NOT_UNIQUE", "message" => "in progress" } }] }
    assert_equal "NOT_UNIQUE", assert_raises(Meeng::Error) { meeng.domains.list }.code
    assert_equal 1, seen.size

    meeng, seen = retrying(max_retries: 0) { LIMITED }
    assert_raises(Meeng::Error) { meeng.domains.list }
    assert_equal 1, seen.size

    meeng, seen = retrying(max_retries: 2) { LIMITED }
    assert_equal "RATE_LIMITED", assert_raises(Meeng::Error) { meeng.domains.list }.code
    assert_equal 3, seen.size
  end

  def test_a_connection_that_fails_is_repeated_for_a_read_only
    meeng, seen = retrying { |n| n == 1 ? raise(Errno::ECONNRESET) : [200, []] }
    meeng.domains.list
    assert_equal 2, seen.size

    meeng, seen = retrying { raise Errno::ECONNRESET }
    assert_raises(Errno::ECONNRESET) { meeng.emails.cancel("em_1") }
    assert_equal 1, seen.size
  end

  def test_sending_an_email_carries_the_key_the_body_and_the_idempotency_key
    meeng, recorder = client { [200, { "emailId" => "em_1" }] }
    sent = meeng.emails.send({ from: "hello@acme.com", to: ["a@x.com", "b@x.com"], subject: "Hi", html: "<p>Hi</p>" }, idempotency_key: "order-1")
    assert_equal "em_1", sent[:emailId]
    request = recorder.seen.first
    assert_equal "POST", request[:method]
    assert_equal "/api/v1/emails", request[:url].path
    assert_equal "Bearer mg_test_key", request[:headers]["Authorization"]
    assert_equal "order-1", request[:headers]["Idempotency-Key"]
    assert_equal "application/json", request[:headers]["Content-Type"]
    assert_equal "meeng-ruby/#{Meeng::VERSION}", request[:headers]["User-Agent"]
    assert_equal({ "from" => "hello@acme.com", "to" => ["a@x.com", "b@x.com"], "subject" => "Hi", "html" => "<p>Hi</p>" }, JSON.parse(request[:body]))
  end

  def test_a_write_with_no_body_of_its_own_sends_an_empty_object
    meeng, recorder = client { [200, { "emailId" => "em_1" }] }
    meeng.emails.cancel("em_1")
    assert_equal "POST", recorder.seen.first[:method]
    assert_equal "/api/v1/emails/em_1/cancel", recorder.seen.first[:url].path
    assert_equal "{}", recorder.seen.first[:body]
  end

  def test_a_send_without_an_idempotency_key_sends_none
    meeng, recorder = client { [200, { "emailId" => "em_1" }] }
    meeng.emails.create({ from: "hello@acme.com", to: "a@x.com", subject: "Hi", text: "Hi" })
    refute recorder.seen.first[:headers].key?("Idempotency-Key")
  end

  def test_a_batch_answers_the_list_of_ids
    meeng, recorder = client { [200, { "data" => [{ "emailId" => "em_1" }, { "emailId" => "em_2" }] }] }
    got = meeng.emails.batch([
                               { from: "hello@acme.com", to: "a@x.com", subject: "One", text: "1" },
                               { from: "hello@acme.com", to: "b@x.com", subject: "Two", text: "2" }
                             ])
    assert_equal "em_2", got[:data][1][:emailId]
    assert_equal "/api/v1/emails/batch", recorder.seen.first[:url].path
    assert_equal 2, JSON.parse(recorder.seen.first[:body]).length
  end

  def test_listing_emails_sends_the_filters_a_list_once_for_each_value
    meeng, recorder = client { [200, { "data" => [], "count" => 0 }] }
    got = meeng.emails.list(page: 2, limit: 10, start_date: "2026-11-01T00:00:00Z", domain_id: %w[1 2])
    assert_equal 0, got[:count]
    request = recorder.seen.first
    assert_equal "GET", request[:method]
    assert_equal "/api/v1/emails", request[:url].path
    pairs = URI.decode_www_form(request[:url].query)
    assert_equal %w[1 2], pairs.select { |name, _| name == "domainId" }.map(&:last)
    assert_equal "2", pairs.to_h["page"]
    assert_equal "10", pairs.to_h["limit"]
    assert_equal "2026-11-01T00:00:00Z", pairs.to_h["startDate"]
    refute pairs.to_h.key?("endDate")
  end

  def test_a_path_parameter_is_escaped_and_a_number_is_accepted
    meeng, recorder = client { [200, { "id" => 7 }] }
    meeng.domains.get(7)
    meeng.emails.get("a/b c")
    assert_equal "/api/v1/domains/7", recorder.seen[0][:url].path
    assert_equal "https://meeng.test/api/v1/emails/a%2Fb%20c", recorder.seen[1][:url].to_s
  end

  def test_a_list_the_api_answers_without_an_object_around_it_is_a_list
    meeng, = client { [200, [{ "id" => "cb_1", "name" => "News" }]] }
    books = meeng.contact_books.list
    assert_equal "News", books.first[:name]
  end

  def test_a_delete_can_carry_a_body
    meeng, recorder = client { [200, { "success" => true, "count" => 2 }] }
    got = meeng.contacts.bulk_delete("cb_1", { contactIds: %w[c1 c2] })
    assert_equal 2, got[:count]
    request = recorder.seen.first
    assert_equal "DELETE", request[:method]
    assert_equal "/api/v1/contactBooks/cb_1/contacts/bulk", request[:url].path
    assert_equal({ "contactIds" => %w[c1 c2] }, JSON.parse(request[:body]))
  end

  def test_upsert_takes_the_contacts_id_in_the_path
    meeng, recorder = client { [200, { "contactId" => "c_1" }] }
    got = meeng.contacts.upsert("cb_1", "c_1", { email: "a@x.com", firstName: "Ana" })
    assert_equal "c_1", got[:contactId]
    assert_equal "PUT", recorder.seen.first[:method]
    assert_equal "/api/v1/contactBooks/cb_1/contacts/c_1", recorder.seen.first[:url].path
  end

  def test_a_refusal_raises_in_every_shape_the_api_writes_it
    [
      [409, { "error" => { "code" => "NOT_UNIQUE", "message" => "Idempotency-Key already used" } }, "NOT_UNIQUE", "Idempotency-Key already used"],
      [403, { "error" => "Forbidden" }, "INTERNAL_SERVER_ERROR", "Forbidden"],
      [400, { "code" => "BAD_REQUEST", "message" => "bad" }, "BAD_REQUEST", "bad"],
      [400, { "success" => false, "error" => { "issues" => [{ "code" => "invalid_type", "path" => ["from"], "message" => "Required" }], "name" => "ZodError" } }, "ZodError", "from: Required"],
      [502, "<html>", "INTERNAL_SERVER_ERROR", "Bad Gateway"]
    ].each do |status, body, code, message|
      meeng, = client { [status, body] }
      refusal = assert_raises(Meeng::Error) { meeng.emails.cancel("em_1") }
      assert_equal status, refusal.status
      assert_equal code, refusal.code
      assert_equal 7, refusal.retry_after
      assert_equal "meeng: #{status} #{code}: #{message}", refusal.message
    end
  end

  def test_an_emails_recipients_may_come_as_one_string_or_a_list
    meeng, = client { [200, { "id" => "em_1", "to" => "a@x.com" }] }
    assert_equal "a@x.com", meeng.emails.get("em_1")[:to]
  end

  def test_the_key_and_the_address_come_from_the_environment
    saved = ENV.values_at("MEENG_API_KEY", "MEENG_BASE_URL")
    ENV["MEENG_API_KEY"] = "mg_env"
    ENV["MEENG_BASE_URL"] = "https://self.hosted.test"
    assert_equal "https://self.hosted.test/api", Meeng::Client.new.url
    ENV.delete("MEENG_API_KEY")
    assert_raises(ArgumentError) { Meeng::Client.new }
  ensure
    ENV["MEENG_API_KEY"], ENV["MEENG_BASE_URL"] = saved
  end

  def test_the_address_must_be_https_except_to_localhost
    ["http://app.meeng.test", "ftp://app.meeng.test", "app.meeng.test", "http://localhost.evil.test"].each do |base|
      assert_raises(ArgumentError, base) { Meeng::Client.new("k", base) }
    end
    ["https://app.meeng.test", "http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000"].each do |base|
      Meeng::Client.new("k", base)
    end
  end

  def test_the_standard_library_transport_never_follows_a_redirect
    server = TCPServer.new("127.0.0.1", 0)
    thread = Thread.new do
      socket = server.accept
      while (line = socket.gets) && line != "\r\n"; end
      socket.write("HTTP/1.1 302 Found\r\nLocation: https://elsewhere.test/\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
      socket.close
    end
    meeng = Meeng::Client.new("k", "http://127.0.0.1:#{server.addr[1]}")
    refusal = assert_raises(Meeng::Error) { meeng.emails.cancel("em_1") }
    assert_equal 302, refusal.status
  ensure
    thread&.join(2)
    server&.close
  end

  SECRET = "whsec_test"
  BODY = '{"id":"evt_1","type":"email.delivered","version":"2026-01-18","createdAt":"2026-11-20T12:00:00Z","teamId":7,"attempt":2,"data":{"id":"em_1","status":"DELIVERED","from":"h@acme.com","to":["a@x.com"],"occurredAt":"2026-11-20T12:00:00Z"}}'

  def sign(timestamp, text = BODY, secret = SECRET)
    "v1=#{OpenSSL::HMAC.hexdigest("SHA256", secret, "#{timestamp}.#{text}")}"
  end

  def test_a_signed_webhook_is_verified_and_read_as_an_event
    timestamp = (Time.now.to_f * 1000).to_i.to_s
    webhooks = Meeng::Client.new("k", "https://meeng.test").webhooks(SECRET)
    headers = { "X-Meeng-Signature" => sign(timestamp), "X-Meeng-Timestamp" => timestamp }
    assert webhooks.verify(BODY, headers: headers)
    event = webhooks.construct_event(BODY, headers: headers)
    assert_equal "email.delivered", event[:type]
    assert_equal ["2026-01-18", 7, 2], event.values_at(:version, :teamId, :attempt)
    assert_equal ["a@x.com"], event[:data][:to]
  end

  def test_the_headers_are_accepted_in_any_case
    timestamp = (Time.now.to_f * 1000).to_i.to_s
    webhooks = Meeng::Webhooks.new(SECRET)
    assert webhooks.verify(BODY, headers: { "x-meeng-signature" => sign(timestamp), "x-meeng-timestamp" => timestamp })
  end

  def test_a_webhook_is_refused_with_the_code_that_says_why
    now = 1_800_000_000_000
    timestamp = now.to_s
    old = (now - (6 * 60 * 1000)).to_s
    webhooks = Meeng::Webhooks.new(SECRET)
    good = { "X-Meeng-Signature" => sign(timestamp), "X-Meeng-Timestamp" => timestamp }
    stale = { "X-Meeng-Signature" => sign(old), "X-Meeng-Timestamp" => old }
    refused = lambda do |headers, body = BODY, **options|
      webhooks.construct_event(body, headers: headers, now: now, **options)
      "ACCEPTED"
    rescue Meeng::WebhookVerificationError => e
      e.code
    end
    assert_equal "ACCEPTED", refused.call(good)
    assert_equal "MISSING_SIGNATURE", refused.call({ "X-Meeng-Timestamp" => timestamp })
    assert_equal "MISSING_TIMESTAMP", refused.call({ "X-Meeng-Signature" => sign(timestamp) })
    assert_equal "INVALID_SIGNATURE_FORMAT", refused.call(good.merge("X-Meeng-Signature" => sign(timestamp)[3..]))
    assert_equal "INVALID_TIMESTAMP", refused.call(good.merge("X-Meeng-Timestamp" => "yesterday"))
    assert_equal "TIMESTAMP_OUT_OF_RANGE", refused.call(stale)
    assert_equal "ACCEPTED", refused.call(stale, tolerance: -1)
    assert_equal "ACCEPTED", refused.call(stale, tolerance: 10 * 60 * 1000)
    assert_equal "SIGNATURE_MISMATCH", refused.call(good, secret: "another")
    assert_equal "SIGNATURE_MISMATCH", refused.call(good, BODY.sub("em_1", "em_2"))
    assert_equal "INVALID_JSON", refused.call({ "X-Meeng-Signature" => sign(timestamp, "not json"), "X-Meeng-Timestamp" => timestamp }, "not json")
  end
end
