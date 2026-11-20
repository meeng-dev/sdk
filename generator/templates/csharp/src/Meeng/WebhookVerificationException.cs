using System;

namespace Meeng;

/// <summary>
/// A request meeng did not sign, signed too long ago, or that is not an event.
/// <see cref="Code"/> is <c>MISSING_SIGNATURE</c>, <c>MISSING_TIMESTAMP</c>,
/// <c>INVALID_SIGNATURE_FORMAT</c>, <c>INVALID_TIMESTAMP</c>,
/// <c>TIMESTAMP_OUT_OF_RANGE</c>, <c>SIGNATURE_MISMATCH</c>, <c>INVALID_BODY</c>
/// or <c>INVALID_JSON</c>.
/// </summary>
public sealed class WebhookVerificationException : Exception
{
    /// <summary>The refusal of a webhook.</summary>
    /// <param name="code">Why it was refused.</param>
    /// <param name="message">What is wrong.</param>
    public WebhookVerificationException(string code, string message)
        : base($"{code}: {message}")
    {
        Code = code;
    }

    /// <summary>Why the webhook was refused: <c>SIGNATURE_MISMATCH</c>…</summary>
    public string Code { get; }
}
