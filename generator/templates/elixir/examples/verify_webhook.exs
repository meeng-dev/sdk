# Checks a webhook the way an endpoint does: the signature and the time are
# verified against the raw body, and then the event is read.
# The request here is made in the program; a real endpoint reads the body and the
# headers of the request it receives, and the secret from its settings.
#
#   mix run examples/verify_webhook.exs

secret = "whsec_exemplo"

body =
  JSON.encode!(%{
    id: "call_1",
    type: "email.delivered",
    version: "2026-01-18",
    createdAt: "2026-11-20T12:00:00Z",
    teamId: 7,
    attempt: 1,
    data: %{id: "em_1", status: "DELIVERED", from: "contato@acme.com.br", to: ["ana@example.com"], occurredAt: "2026-11-20T12:00:00Z"}
  })

timestamp = Integer.to_string(System.os_time(:millisecond))
signature = :crypto.mac(:hmac, :sha256, secret, timestamp <> "." <> body) |> Base.encode16(case: :lower)
headers = [{"x-meeng-signature", "v1=" <> signature}, {"x-meeng-timestamp", timestamp}]

case Meeng.construct_webhook_event(secret, headers, body) do
  {:ok, %{"type" => "email.delivered", "data" => data}} ->
    IO.puts("e-mail #{data["id"]} entregue a #{Enum.join(data["to"], ", ")}")

  {:ok, %{"type" => type}} ->
    IO.puts("evento #{type}")

  {:error, error} ->
    IO.puts(:stderr, "#{error.wire_code}: #{error.message}")
    System.halt(1)
end
