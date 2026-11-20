# Schedules an e-mail, moves it to another time and cancels it.
#
#   MEENG_API_KEY=mg_xxx mix run examples/schedule_email.exs

meeng = Meeng.new()
in_hours = fn hours -> DateTime.utc_now() |> DateTime.add(hours * 3600) |> DateTime.truncate(:second) |> DateTime.to_iso8601() end

with {:ok, %{"emailId" => id}} <-
       Meeng.Emails.send(meeng, %{
         from: "Acme <contato@acme.com.br>",
         to: "ana@example.com",
         subject: "Lembrete: sua reunião é amanhã",
         html: "<p>Nos vemos às 10h.</p>",
         scheduledAt: in_hours.(24)
       }),
     _ = IO.puts("agendado: #{id}"),
     {:ok, _} <- Meeng.Emails.update(meeng, id, %{scheduledAt: in_hours.(48)}),
     _ = IO.puts("remarcado"),
     {:ok, _} <- Meeng.Emails.cancel(meeng, id) do
  IO.puts("cancelado")
else
  {:error, error} ->
    IO.puts(:stderr, Exception.message(error))
    System.halt(1)
end
