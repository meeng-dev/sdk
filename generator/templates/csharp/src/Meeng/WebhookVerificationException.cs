using System;

namespace Meeng;

/// <summary>A request meeng did not sign, or signed too long ago.</summary>
public sealed class WebhookVerificationException : Exception
{
    /// <summary>The refusal of a webhook.</summary>
    public WebhookVerificationException()
        : base("meeng: webhook signature does not verify")
    {
    }
}
