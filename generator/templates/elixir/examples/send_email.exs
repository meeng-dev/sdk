# Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client at
# another app, the address from MEENG_BASE_URL.
#
#   MEENG_API_KEY=mg_xxx mix run examples/send_email.exs

meeng = Meeng.new()

result =
  Meeng.Emails.send(
    meeng,
    %{
      from: "Acme <contato@acme.com.br>",
      to: "ana@example.com",
      subject: "Bem-vinda à Acme",
      html: "<p>Olá, Ana! Sua conta está pronta.</p>",
      text: "Olá, Ana! Sua conta está pronta."
    },
    # A key makes the send safe to repeat: the same key with the same body
    # answers the first result again, and nothing is sent twice.
    idempotency_key: "boas-vindas-ana-2026-11-20"
  )

case result do
  {:ok, %{"emailId" => id}} ->
    IO.puts("e-mail na fila: #{id}")

  {:error, error} ->
    IO.puts(:stderr, Exception.message(error))
    System.halt(1)
end
