defmodule Meeng.Client do
  @moduledoc """
  meeng's API with one team's API key.

      client = Meeng.Client.new("mg_123")
      {:ok, email} = Meeng.Emails.get(client, "em_1")
  """

  @derive {Inspect, except: [:api_key]}
  @enforce_keys [:api_key]
  defstruct [
    :api_key,
    base_url: nil,
    timeout: 30_000,
    max_retries: 2,
    retry_delay: 500,
    transport: &Meeng.Client.httpc/5
  ]

  # The longest, in milliseconds, a call waits before a repetition.
  @max_retry_delay 30_000

  @typedoc """
  Sends one request: `(method, url, headers, body, timeout_ms)` answers
  `{:ok, {status, headers_in_lower_case, body}}` or `{:error, reason}`.
  """
  @type transport ::
          (String.t(), String.t(), [{String.t(), String.t()}], binary() | nil, timeout() ->
             {:ok, {non_neg_integer(), [{String.t(), String.t()}], binary()}} | {:error, term()})

  @type t :: %__MODULE__{
          api_key: String.t(),
          base_url: String.t(),
          timeout: timeout(),
          max_retries: non_neg_integer(),
          retry_delay: non_neg_integer(),
          transport: transport()
        }

  @doc """
  A client for the team `api_key` belongs to; `MEENG_API_KEY` when it is nil.

  Options: `:base_url` (the address of the app, without `/api/v1`, such as a
  self-hosted one; `MEENG_BASE_URL` when left out. It must be https, or http only
  for localhost, 127.0.0.0/8 and ::1, or `ArgumentError`), `:timeout` in
  milliseconds (30 seconds), `:transport` (see `t:transport/0`), `:max_retries`
  and `:retry_delay`.

  A call that may be repeated is repeated up to `:max_retries` times (2 by
  default; 0 sends every call once) after the first attempt. It is repeated when
  the API answers 429, which means it was not processed, so any call may be; and
  when the API answers 500, 502, 503 or 504, or the connection fails, only if
  repeating it cannot do the work twice: a read, a PUT, a DELETE, or any call that
  carries an idempotency key. Any other refusal, a 409 included, is never
  repeated. The first repetition waits `:retry_delay` milliseconds (500), each
  next one twice as long, and a Retry-After from the API replaces the wait.
  """
  @spec new(String.t() | nil, keyword()) :: t()
  def new(api_key \\ nil, opts \\ []) do
    api_key = api_key || System.get_env("MEENG_API_KEY")

    unless is_binary(api_key) and api_key != "" do
      raise ArgumentError,
            "Missing API key. Pass it to Meeng.new(\"mg_123\") or set MEENG_API_KEY"
    end

    base_url =
      (Keyword.get(opts, :base_url) || System.get_env("MEENG_BASE_URL") ||
         Meeng.default_base_url())
      |> String.trim_trailing("/")

    check_base_url!(base_url)

    %__MODULE__{
      api_key: api_key,
      base_url: base_url <> Meeng.api_prefix(),
      timeout: Keyword.get(opts, :timeout, 30_000),
      max_retries: max(Keyword.get(opts, :max_retries, 2), 0),
      retry_delay: max(Keyword.get(opts, :retry_delay, 500), 0),
      transport: Keyword.get(opts, :transport, &__MODULE__.httpc/5)
    }
  end

  @doc """
  Calls the API and answers its JSON, decoded.

  Options: `:query`, `:body` and `:idempotency_key`.
  """
  @spec request(t(), String.t(), String.t(), keyword()) ::
          {:ok, term()} | {:error, Meeng.Error.t()}
  def request(%__MODULE__{} = client, method, path, opts \\ []) do
    headers =
      [
        {"Authorization", "Bearer " <> client.api_key},
        {"Accept", "application/json"},
        {"User-Agent", "meeng-elixir/" <> Meeng.version()}
      ]
      |> put_header("Content-Type", opts[:body] && "application/json")
      |> put_header("Idempotency-Key", opts[:idempotency_key])

    body = if opts[:body], do: JSON.encode!(body(opts[:body]))
    url = client.base_url <> path <> query(opts[:query])

    may_repeat? =
      method in ["GET", "HEAD", "PUT", "DELETE"] or opts[:idempotency_key] not in [nil, ""]

    send_request(client, {method, url, headers, body}, may_repeat?, 0)
  end

  defp send_request(client, {method, url, headers, body} = request, may_repeat?, attempt) do
    again? = attempt < client.max_retries

    case client.transport.(method, url, headers, body, client.timeout) do
      {:ok, {status, response_headers, _raw}}
      when again? and (status == 429 or (may_repeat? and status in [500, 502, 503, 504])) ->
        pause(client, attempt, retry_after(response_headers))
        send_request(client, request, may_repeat?, attempt + 1)

      {:ok, {status, response_headers, raw}} when status >= 300 ->
        {:error, Meeng.Error.from_response(status, response_headers, raw)}

      {:ok, {_status, _headers, raw}} ->
        case JSON.decode(raw) do
          {:ok, data} ->
            {:ok, data}

          _ ->
            {:error,
             %Meeng.Error{
               status: nil,
               code: "invalid_response",
               message: "the answer is not JSON"
             }}
        end

      {:error, _reason} when again? and may_repeat? ->
        # The connection failed or timed out: the call may or may not have arrived.
        pause(client, attempt, nil)
        send_request(client, request, may_repeat?, attempt + 1)

      {:error, reason} ->
        {:error, %Meeng.Error{status: nil, code: "transport", message: inspect(reason)}}
    end
  end

  # Sleeps before repetition number attempt + 1: the API's Retry-After, or the
  # base delay doubled each time and spread over its upper half.
  defp pause(client, attempt, retry_after) do
    millis =
      case retry_after do
        seconds when is_number(seconds) and seconds > 0 ->
          round(seconds * 1000)

        _ ->
          delay = min(client.retry_delay * Integer.pow(2, min(attempt, 10)), @max_retry_delay)
          div(delay, 2) + :rand.uniform(div(delay, 2) + 1) - 1
      end

    Process.sleep(min(millis, @max_retry_delay))
  end

  defp retry_after(headers) do
    case List.keyfind(headers, "retry-after", 0) do
      {_name, value} ->
        case Float.parse(to_string(value)) do
          {seconds, _rest} -> seconds
          :error -> nil
        end

      nil ->
        nil
    end
  end

  @doc false
  # A body as JSON writes it: a keyword list is its map, a list of them walked.
  @spec body(term()) :: term()
  def body([_ | _] = list) do
    if Keyword.keyword?(list), do: Map.new(list), else: Enum.map(list, &body/1)
  end

  def body(other), do: other

  @doc false
  # A path parameter, escaped.
  @spec segment(String.t() | integer()) :: String.t()
  def segment(value), do: value |> to_string() |> URI.encode(&URI.char_unreserved?/1)

  # The base URL must be https; http is accepted only for localhost,
  # 127.0.0.0/8 and ::1.
  defp check_base_url!(base_url) do
    %URI{scheme: scheme, host: host} = URI.parse(base_url)
    scheme = String.downcase(scheme || "")
    host = String.downcase(host || "")

    loopback? =
      host == "localhost" or
        match?(
          {:ok, {127, _, _, _}},
          :inet.parse_address(String.to_charlist(host))
        ) or
        match?(
          {:ok, {0, 0, 0, 0, 0, 0, 0, 1}},
          :inet.parse_address(String.to_charlist(host))
        )

    unless scheme == "https" or (scheme == "http" and loopback?) do
      raise ArgumentError,
            "meeng: the base URL must be https (http is accepted only for localhost, 127.0.0.0/8 and ::1), so the API key is never sent in clear text"
    end
  end

  defp put_header(headers, _name, value) when value in [nil, false, ""], do: headers
  defp put_header(headers, name, value), do: headers ++ [{name, value}]

  defp query(nil), do: ""

  defp query(values) do
    pairs =
      Enum.flat_map(values, fn
        # A list is sent once for each value: name=a&name=b.
        {name, list} when is_list(list) -> Enum.map(list, &{name, &1})
        pair -> [pair]
      end)

    case Enum.reject(pairs, fn {_name, value} -> value in [nil, ""] end) do
      [] -> ""
      kept -> "?" <> URI.encode_query(kept)
    end
  end

  @doc """
  The standard library's HTTP, `:httpc`, so the SDK depends on nothing. HTTPS
  verifies the server against the operating system's certificates.
  """
  @spec httpc(String.t(), String.t(), [{String.t(), String.t()}], binary() | nil, timeout()) ::
          {:ok, {non_neg_integer(), [{String.t(), String.t()}], binary()}} | {:error, term()}
  def httpc(method, url, headers, body, timeout) do
    {:ok, _started} = Application.ensure_all_started([:inets, :ssl])
    {content_type, headers} = List.keytake(headers, "Content-Type", 0) || {nil, headers}

    headers =
      for {name, value} <- headers, do: {String.to_charlist(name), String.to_charlist(value)}

    url = String.to_charlist(url)

    request =
      case {body, content_type} do
        {nil, _} when method in ["GET", "DELETE"] -> {url, headers}
        {nil, _} -> {url, headers, ~c"", ""}
        {body, {_name, type}} -> {url, headers, String.to_charlist(type), body}
      end

    options = [timeout: timeout, connect_timeout: timeout, autoredirect: false, ssl: ssl(url)]

    case :httpc.request(verb(method), request, options, body_format: :binary) do
      {:ok, {{_version, status, _reason}, response_headers, raw}} ->
        {:ok,
         {status, for({name, value} <- response_headers, do: {to_string(name), to_string(value)}),
          raw}}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp verb("GET"), do: :get
  defp verb("POST"), do: :post
  defp verb("PUT"), do: :put
  defp verb("PATCH"), do: :patch
  defp verb("DELETE"), do: :delete

  defp ssl(~c"https://" ++ _rest) do
    [
      verify: :verify_peer,
      cacerts: :public_key.cacerts_get(),
      customize_hostname_check: [match_fun: :public_key.pkix_verify_hostname_match_fun(:https)]
    ]
  end

  defp ssl(_url), do: []
end
