using System;
using System.Globalization;
using System.Text.Json;

namespace Meeng;

/// <summary>
/// A request the API refused. <see cref="Code"/> is the API's, such as
/// <c>NOT_UNIQUE</c>, <c>FORBIDDEN</c> or <c>NOT_FOUND</c>; it is
/// <c>INTERNAL_SERVER_ERROR</c> when the answer carried none.
/// <see cref="RetryAfter"/> is how many seconds to wait on a 429.
/// </summary>
public sealed class MeengException : Exception
{
    /// <summary>A refusal.</summary>
    /// <param name="status">The HTTP status.</param>
    /// <param name="code">The API's error code.</param>
    /// <param name="apiMessage">The API's message.</param>
    /// <param name="retryAfter">Seconds to wait before trying again, or null.</param>
    public MeengException(int status, string code, string apiMessage, int? retryAfter = null)
        : base(apiMessage.Length > 0 ? $"meeng: {status} {code}: {apiMessage}" : $"meeng: {status} {code}")
    {
        Status = status;
        Code = code;
        ApiMessage = apiMessage;
        RetryAfter = retryAfter;
    }

    /// <summary>The HTTP status.</summary>
    public int Status { get; }

    /// <summary>The API's error code: NOT_UNIQUE, FORBIDDEN…</summary>
    public string Code { get; }

    /// <summary>The API's message.</summary>
    public string ApiMessage { get; }

    /// <summary>On a 429, how many seconds to wait; null otherwise.</summary>
    public int? RetryAfter { get; }

    /// <summary>
    /// The refusal a response carries: the answer's <c>{"error": {"code",
    /// "message"}}</c>, <c>{"error": "text"}</c> or the error itself. A body that
    /// is not the API's JSON is still a refusal.
    /// </summary>
    internal static MeengException FromResponse(int status, string? retryAfter, byte[] body)
    {
        string code = "", message = "";
        try
        {
            using var document = JsonDocument.Parse(body);
            var root = document.RootElement;
            if (root.ValueKind == JsonValueKind.Object)
            {
                var found = root.TryGetProperty("error", out var wrapped) ? wrapped : root;
                if (found.ValueKind == JsonValueKind.String)
                {
                    message = found.GetString()!;
                }
                else if (found.ValueKind == JsonValueKind.Object)
                {
                    code = Text(found, "code");
                    message = Text(found, "message");
                }
            }
        }
        catch (JsonException)
        {
            // A refusal without a JSON body is still a refusal.
        }
        if (code.Length == 0)
        {
            code = "INTERNAL_SERVER_ERROR";
        }
        if (message.Length == 0)
        {
            message = Reason(status);
        }
        int? seconds = int.TryParse(retryAfter, NumberStyles.None, CultureInfo.InvariantCulture, out var parsed) ? parsed : null;
        return new MeengException(status, code, message, seconds);
    }

    private static string Text(JsonElement root, string name) =>
        root.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String ? value.GetString()! : "";

    private static string Reason(int status) => status switch
    {
        400 => "Bad Request",
        401 => "Unauthorized",
        403 => "Forbidden",
        404 => "Not Found",
        409 => "Conflict",
        422 => "Unprocessable Entity",
        429 => "Too Many Requests",
        500 => "Internal Server Error",
        502 => "Bad Gateway",
        503 => "Service Unavailable",
        _ => "Error",
    };
}
