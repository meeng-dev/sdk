# Sends up to 100 e-mails in one request.
#
#   MEENG_API_KEY=mg_xxx mix run examples/send_batch.exs

meeng = Meeng.new()
people = [{"Ana", "ana@example.com"}, {"Bruno", "bruno@example.com"}, {"Carla", "carla@example.com"}]

batch =
  for {name, email} <- people do
    %{
      from: "Acme <contato@acme.com.br>",
      to: email,
      subject: "Seu extrato de novembro",
      html: "<p>Olá, #{name}! Seu extrato está disponível.</p>"
    }
  end

case Meeng.Emails.batch(meeng, batch, idempotency_key: "extrato-2026-11") do
  {:ok, %{"data" => emails}} ->
    emails |> Enum.with_index(1) |> Enum.each(fn {%{"emailId" => id}, n} -> IO.puts("#{n}: #{id}") end)

  {:error, error} ->
    IO.puts(:stderr, Exception.message(error))
    System.halt(1)
end
