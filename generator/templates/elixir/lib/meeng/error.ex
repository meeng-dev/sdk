defmodule Meeng.Error do
  @moduledoc """
  A request the API refused.

  `code` is the API's, such as `NOT_UNIQUE`, `FORBIDDEN` or `NOT_FOUND`; it is
  `INTERNAL_SERVER_ERROR` when the answer carried none. `status` is the HTTP
  status and `retry_after`, on a 429, how many seconds to wait.

  A request that never got an answer has no `status` and the code `transport`.
  """

  defexception [:status, code: "", message: "", retry_after: nil]

  @type t :: %__MODULE__{
          status: non_neg_integer() | nil,
          code: String.t(),
          message: String.t(),
          retry_after: non_neg_integer() | nil
        }

  @impl true
  def message(%__MODULE__{status: status, code: code, message: ""}),
    do: "meeng: #{status} #{code}"

  def message(%__MODULE__{status: status, code: code, message: message}),
    do: "meeng: #{status} #{code}: #{message}"

  @reasons %{
    400 => "Bad Request",
    401 => "Unauthorized",
    403 => "Forbidden",
    404 => "Not Found",
    409 => "Conflict",
    422 => "Unprocessable Entity",
    429 => "Too Many Requests",
    500 => "Internal Server Error",
    502 => "Bad Gateway",
    503 => "Service Unavailable"
  }

  @doc """
  The error a response with a status of 300 or more is: the answer's
  `{"error": {"code", "message"}}`, `{"error": "text"}` or the error itself.
  """
  @spec from_response(non_neg_integer(), [{String.t(), String.t()}], binary()) :: t()
  def from_response(status, headers, body) do
    {code, message} =
      case JSON.decode(body) do
        {:ok, %{} = data} -> found(Map.get(data, "error", data))
        # A refusal without a JSON body is still a refusal.
        _ -> {nil, nil}
      end

    %__MODULE__{
      status: status,
      code: if(is_binary(code) and code != "", do: code, else: "INTERNAL_SERVER_ERROR"),
      message:
        if(is_binary(message) and message != "",
          do: message,
          else: Map.get(@reasons, status, "Error")
        ),
      retry_after: retry_after(headers)
    }
  end

  defp found(text) when is_binary(text), do: {nil, text}
  # A refused validation answers {"success": false, "error": {"issues", "name"}},
  # with no code of its own.
  defp found(%{} = data) do
    code = if is_binary(data["code"]) and data["code"] != "", do: data["code"], else: data["name"]

    message =
      case {data["message"], data["issues"]} do
        {message, _} when is_binary(message) and message != "" -> message
        {_, [%{"message" => text} = first | _]} when is_binary(text) -> where(first) <> text
        _ -> nil
      end

    {code, message}
  end

  defp found(_other), do: {nil, nil}

  defp where(%{"path" => [_ | _] = path}), do: Enum.map_join(path, ".", &to_string/1) <> ": "
  defp where(_issue), do: ""

  defp retry_after(headers) do
    with {_name, value} <- List.keyfind(headers, "retry-after", 0),
         {seconds, ""} <- Integer.parse(value) do
      seconds
    else
      _ -> nil
    end
  end
end
