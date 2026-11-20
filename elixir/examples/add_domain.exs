# Adds a sending domain, prints the DNS records to publish and asks for them to
# be checked.
#
#   MEENG_API_KEY=mg_xxx mix run examples/add_domain.exs

meeng = Meeng.new()

fail = fn error ->
  IO.puts(:stderr, Exception.message(error))
  System.halt(1)
end

case Meeng.Domains.create(meeng, %{name: "acme.com.br", region: "us-east-1"}) do
  {:ok, domain} ->
    IO.puts("publique estes registros no DNS de #{domain["name"]}")

    for record <- domain["dnsRecords"] do
      IO.puts("  #{String.pad_trailing(record["type"], 5)} #{String.pad_trailing(record["name"], 40)} #{record["value"]}")
    end

    with {:ok, _} <- Meeng.Domains.verify(meeng, domain["id"]),
         {:ok, checked} <- Meeng.Domains.get(meeng, domain["id"]) do
      IO.puts("situação: #{checked["status"]}")
    else
      {:error, error} -> fail.(error)
    end

  {:error, error} ->
    fail.(error)
end
