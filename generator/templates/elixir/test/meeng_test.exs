# meeng's Elixir SDK tests, with nothing to install: mix test

defmodule MeengTest do
  use ExUnit.Case, async: true

  # A client whose transport sends each request to the test and answers what
  # `answer` says.
  defp client(answer) do
    test = self()

    transport = fn method, url, headers, body, _timeout ->
      request = %{method: method, url: URI.parse(url), headers: Map.new(headers), body: body}
      send(test, {:request, request})
      {status, payload} = answer.(request)
      raw = if is_binary(payload), do: payload, else: JSON.encode!(payload)
      {:ok, {status, [{"retry-after", "7"}], raw}}
    end

    Meeng.new("mg_test_key", base_url: "https://meeng.test/", transport: transport)
  end

  test "sending an email carries the key, the body and the idempotency key" do
    meeng = client(fn _request -> {200, %{"emailId" => "em_1"}} end)

    assert {:ok, %{"emailId" => "em_1"}} =
             Meeng.Emails.send(
               meeng,
               %{
                 from: "hello@acme.com",
                 to: ["a@x.com", "b@x.com"],
                 subject: "Hi",
                 html: "<p>Hi</p>"
               },
               idempotency_key: "order-1"
             )

    assert_received {:request, seen}
    assert seen.method == "POST"
    assert seen.url.path == "/api/v1/emails"
    assert seen.headers["Authorization"] == "Bearer mg_test_key"
    assert seen.headers["Idempotency-Key"] == "order-1"
    assert seen.headers["Content-Type"] == "application/json"
    assert seen.headers["User-Agent"] == "meeng-elixir/" <> Meeng.version()

    assert JSON.decode!(seen.body) == %{
             "from" => "hello@acme.com",
             "to" => ["a@x.com", "b@x.com"],
             "subject" => "Hi",
             "html" => "<p>Hi</p>"
           }
  end

  test "a write with no body of its own sends an empty object" do
    meeng = client(fn _request -> {200, %{"emailId" => "em_1"}} end)
    assert {:ok, _} = Meeng.Emails.cancel(meeng, "em_1")
    assert_received {:request, seen}
    assert seen.method == "POST"
    assert seen.url.path == "/api/v1/emails/em_1/cancel"
    assert seen.body == "{}"
  end

  test "a send without an idempotency key sends none, and a keyword list is a body" do
    meeng = client(fn _request -> {200, %{"emailId" => "em_1"}} end)

    assert {:ok, _} =
             Meeng.Emails.create(meeng, from: "hello@acme.com", to: "a@x.com", subject: "Hi")

    assert_received {:request, seen}
    refute Map.has_key?(seen.headers, "Idempotency-Key")
    assert JSON.decode!(seen.body)["to"] == "a@x.com"
  end

  test "a batch answers the list of ids" do
    meeng =
      client(fn _request ->
        {200, %{"data" => [%{"emailId" => "em_1"}, %{"emailId" => "em_2"}]}}
      end)

    assert {:ok, %{"data" => [_, %{"emailId" => "em_2"}]}} =
             Meeng.Emails.batch(meeng, [
               %{from: "hello@acme.com", to: "a@x.com", subject: "One", text: "1"},
               %{from: "hello@acme.com", to: "b@x.com", subject: "Two", text: "2"}
             ])

    assert_received {:request, seen}
    assert seen.url.path == "/api/v1/emails/batch"
    assert length(JSON.decode!(seen.body)) == 2
  end

  test "listing emails sends the filters, a list once for each value" do
    meeng = client(fn _request -> {200, %{"data" => [], "count" => 0}} end)

    assert {:ok, %{"count" => 0}} =
             Meeng.Emails.list(meeng,
               page: 2,
               limit: 10,
               start_date: "2026-11-01T00:00:00Z",
               domain_id: ["1", "2"]
             )

    assert_received {:request, seen}
    assert seen.method == "GET"
    assert seen.url.path == "/api/v1/emails"
    pairs = seen.url.query |> URI.query_decoder() |> Enum.to_list()
    assert for({"domainId", id} <- pairs, do: id) == ["1", "2"]
    assert List.keyfind(pairs, "page", 0) == {"page", "2"}
    assert List.keyfind(pairs, "limit", 0) == {"limit", "10"}
    assert List.keyfind(pairs, "startDate", 0) == {"startDate", "2026-11-01T00:00:00Z"}
    refute List.keyfind(pairs, "endDate", 0)
  end

  test "a path parameter is escaped and a number is accepted" do
    meeng = client(fn _request -> {200, %{"id" => 7}} end)
    assert {:ok, _} = Meeng.Domains.get(meeng, 7)
    assert {:ok, _} = Meeng.Emails.get(meeng, "a/b c")
    assert_received {:request, domain}
    assert domain.url.path == "/api/v1/domains/7"
    assert_received {:request, email}
    assert to_string(%{email.url | query: nil}) == "https://meeng.test/api/v1/emails/a%2Fb%20c"
  end

  test "a list the API answers without an object around it is a list" do
    meeng = client(fn _request -> {200, [%{"id" => "cb_1", "name" => "News"}]} end)
    assert {:ok, [%{"name" => "News"}]} = Meeng.ContactBooks.list(meeng)
  end

  test "a delete can carry a body" do
    meeng = client(fn _request -> {200, %{"success" => true, "count" => 2}} end)

    assert {:ok, %{"count" => 2}} =
             Meeng.Contacts.bulk_delete(meeng, "cb_1", %{contactIds: ["c1", "c2"]})

    assert_received {:request, seen}
    assert seen.method == "DELETE"
    assert seen.url.path == "/api/v1/contactBooks/cb_1/contacts/bulk"
    assert JSON.decode!(seen.body) == %{"contactIds" => ["c1", "c2"]}
  end

  test "upsert takes the contact's id in the path" do
    meeng = client(fn _request -> {200, %{"contactId" => "c_1"}} end)

    assert {:ok, %{"contactId" => "c_1"}} =
             Meeng.Contacts.upsert(meeng, "cb_1", "c_1", %{email: "a@x.com", firstName: "Ana"})

    assert_received {:request, seen}
    assert seen.method == "PUT"
    assert seen.url.path == "/api/v1/contactBooks/cb_1/contacts/c_1"
  end

  test "a refusal is an error, in every shape the API writes it" do
    cases = [
      {409, %{"error" => %{"code" => "NOT_UNIQUE", "message" => "Idempotency-Key already used"}},
       "NOT_UNIQUE", "Idempotency-Key already used"},
      {403, %{"error" => "Forbidden"}, "INTERNAL_SERVER_ERROR", "Forbidden"},
      {400, %{"code" => "BAD_REQUEST", "message" => "bad"}, "BAD_REQUEST", "bad"},
      {400,
       %{
         "success" => false,
         "error" => %{
           "issues" => [%{"code" => "invalid_type", "path" => ["from"], "message" => "Required"}],
           "name" => "ZodError"
         }
       }, "ZodError", "from: Required"},
      {502, "<html>", "INTERNAL_SERVER_ERROR", "Bad Gateway"}
    ]

    for {status, body, code, message} <- cases do
      meeng = client(fn _request -> {status, body} end)

      assert {:error, %Meeng.Error{status: ^status, code: ^code, retry_after: 7} = error} =
               Meeng.Emails.cancel(meeng, "em_1")

      assert Exception.message(error) == "meeng: #{status} #{code}: #{message}"
    end
  end

  test "an email's recipients may come as one string or a list" do
    meeng = client(fn _request -> {200, %{"id" => "em_1", "to" => "a@x.com"}} end)
    assert {:ok, %{"to" => "a@x.com"}} = Meeng.Emails.get(meeng, "em_1")
  end

  test "the key and the address come from the environment" do
    System.put_env("MEENG_API_KEY", "mg_env")
    System.put_env("MEENG_BASE_URL", "https://self.hosted.test")

    try do
      meeng = Meeng.new()
      assert meeng.api_key == "mg_env"
      assert meeng.base_url == "https://self.hosted.test/api"
      System.delete_env("MEENG_API_KEY")
      assert_raise ArgumentError, fn -> Meeng.new() end
    after
      System.delete_env("MEENG_API_KEY")
      System.delete_env("MEENG_BASE_URL")
    end
  end

  test "the address must be https, except to localhost" do
    for base <- [
          "http://app.meeng.test",
          "ftp://app.meeng.test",
          "app.meeng.test",
          "http://localhost.evil.test"
        ] do
      assert_raise ArgumentError, fn -> Meeng.new("k", base_url: base) end
    end

    for base <- [
          "https://app.meeng.test",
          "http://localhost:3000",
          "http://127.0.0.1:3000",
          "http://[::1]:3000"
        ] do
      assert %Meeng.Client{} = Meeng.new("k", base_url: base)
    end
  end

  # A server that reads one request, sends it to the test and answers `reply`.
  defp serve(reply) do
    {:ok, listener} =
      :gen_tcp.listen(0, [:binary, active: false, ip: {127, 0, 0, 1}, reuseaddr: true])

    {:ok, port} = :inet.port(listener)
    test = self()

    spawn_link(fn ->
      {:ok, socket} = :gen_tcp.accept(listener)
      {:ok, data} = :gen_tcp.recv(socket, 0, 5000)
      data = read_body(socket, data)
      send(test, {:raw_request, data})
      :ok = :gen_tcp.send(socket, reply)
      :gen_tcp.close(socket)
      :gen_tcp.close(listener)
    end)

    port
  end

  defp read_body(socket, data) do
    case String.split(data, "\r\n\r\n", parts: 2) do
      [head, body] ->
        length =
          case Regex.run(~r/content-length: (\d+)/i, head) do
            [_, length] -> String.to_integer(length)
            _ -> 0
          end

        if byte_size(body) < length do
          {:ok, more} = :gen_tcp.recv(socket, 0, 5000)
          read_body(socket, data <> more)
        else
          data
        end

      _ ->
        {:ok, more} = :gen_tcp.recv(socket, 0, 5000)
        read_body(socket, data <> more)
    end
  end

  test "the standard library transport sends a body on a delete" do
    answer = ~s({"success":true,"count":2})

    port =
      serve(
        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: #{byte_size(answer)}\r\nConnection: close\r\n\r\n" <>
          answer
      )

    meeng = Meeng.new("mg_test_key", base_url: "http://127.0.0.1:#{port}")

    assert {:ok, %{"count" => 2}} =
             Meeng.Contacts.bulk_delete(meeng, "cb_1", %{contactIds: ["c1"]})

    assert_received {:raw_request, raw}
    assert raw =~ "DELETE /api/v1/contactBooks/cb_1/contacts/bulk HTTP/1.1"
    assert raw =~ ~s({"contactIds":["c1"]})
  end

  test "the standard library transport never follows a redirect" do
    port =
      serve(
        "HTTP/1.1 302 Found\r\nLocation: https://elsewhere.test/\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"
      )

    meeng = Meeng.new("mg_test_key", base_url: "http://127.0.0.1:#{port}")
    assert {:error, %Meeng.Error{status: 302}} = Meeng.Emails.cancel(meeng, "em_1")
  end

  @secret "whsec_test"
  @body ~s({"id":"evt_1","type":"email.delivered","version":"2026-01-18","createdAt":"2026-11-20T12:00:00Z","teamId":7,"attempt":2,"data":{"id":"em_1","status":"DELIVERED","from":"h@acme.com","to":["a@x.com"],"occurredAt":"2026-11-20T12:00:00Z"}})

  defp sign(timestamp, body \\ @body, secret \\ @secret) do
    "v1=" <>
      Base.encode16(:crypto.mac(:hmac, :sha256, secret, [timestamp, ".", body]), case: :lower)
  end

  test "a signed webhook is verified and read as an event" do
    timestamp = Integer.to_string(System.os_time(:millisecond))
    headers = %{"X-Meeng-Signature" => sign(timestamp), "X-Meeng-Timestamp" => timestamp}
    assert Meeng.Webhooks.verify(@secret, headers, @body)
    assert {:ok, event} = Meeng.Webhooks.construct_event(@secret, headers, @body)
    assert event["type"] == "email.delivered"
    assert {event["version"], event["teamId"], event["attempt"]} == {"2026-01-18", 7, 2}
    assert event["data"]["to"] == ["a@x.com"]
  end

  test "the useSend headers are accepted as well, in any case" do
    timestamp = Integer.to_string(System.os_time(:millisecond))
    headers = [{"x-usesend-signature", sign(timestamp)}, {"X-UseSend-Timestamp", timestamp}]
    assert Meeng.verify_webhook(@secret, headers, @body)
  end

  test "a webhook is refused with the code that says why" do
    now = 1_800_000_000_000
    timestamp = Integer.to_string(now)
    old = Integer.to_string(now - 6 * 60 * 1000)
    good = %{"X-Meeng-Signature" => sign(timestamp), "X-Meeng-Timestamp" => timestamp}
    stale = %{"X-Meeng-Signature" => sign(old), "X-Meeng-Timestamp" => old}

    refused = fn headers, body, secret, opts ->
      case Meeng.Webhooks.construct_event(secret, headers, body, [now: now] ++ opts) do
        {:ok, _event} -> :accepted
        {:error, error} -> error.wire_code
      end
    end

    assert refused.(good, @body, @secret, []) == :accepted

    assert refused.(%{"X-Meeng-Timestamp" => timestamp}, @body, @secret, []) ==
             "MISSING_SIGNATURE"

    assert refused.(%{"X-Meeng-Signature" => sign(timestamp)}, @body, @secret, []) ==
             "MISSING_TIMESTAMP"

    assert refused.(
             %{good | "X-Meeng-Signature" => String.slice(sign(timestamp), 3..-1//1)},
             @body,
             @secret,
             []
           ) ==
             "INVALID_SIGNATURE_FORMAT"

    assert refused.(%{good | "X-Meeng-Timestamp" => "yesterday"}, @body, @secret, []) ==
             "INVALID_TIMESTAMP"

    assert refused.(stale, @body, @secret, []) == "TIMESTAMP_OUT_OF_RANGE"
    assert refused.(stale, @body, @secret, tolerance: -1) == :accepted
    assert refused.(stale, @body, @secret, tolerance: 10 * 60 * 1000) == :accepted
    assert refused.(good, @body, "another", []) == "SIGNATURE_MISMATCH"

    assert refused.(good, String.replace(@body, "em_1", "em_2"), @secret, []) ==
             "SIGNATURE_MISMATCH"

    assert refused.(
             %{
               "X-Meeng-Signature" => sign(timestamp, "not json"),
               "X-Meeng-Timestamp" => timestamp
             },
             "not json",
             @secret,
             []
           ) == "INVALID_JSON"
  end
end
