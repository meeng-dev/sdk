using System.Text.Json;

namespace Meeng;

/// <summary>
/// A webhook meeng sent, verified: one record a kind of data
/// (<see cref="EmailWebhookEvent"/> carries an <see cref="EmailBasePayload"/>…), or
/// <see cref="UnknownWebhookEvent"/> for a type this SDK does not know yet.
/// Within a record, <see cref="Type"/> says what happened.
/// </summary>
/// <param name="Id">The id of the delivery: the same on every attempt, so a receiver can deduplicate by it.</param>
/// <param name="Type">What happened: <c>email.delivered</c>…</param>
/// <param name="CreatedAt">When the event was created, RFC 3339.</param>
public abstract record WebhookEvent(string Id, string Type, string CreatedAt)
{
    /// <summary>The version of the payload's shape, such as <c>2026-01-18</c>; null when an older server sent none.</summary>
    public string? Version { get; init; }

    /// <summary>The team the event belongs to; null when an older server sent none.</summary>
    public long? TeamId { get; init; }

    /// <summary>Which delivery attempt this is, from 1 to 6; null when an older server sent none.</summary>
    public int? Attempt { get; init; }
}

/// <summary>An event of a type this SDK does not know yet, its data as it was sent.</summary>
/// <param name="Id">The event's id.</param>
/// <param name="Type">What happened.</param>
/// <param name="CreatedAt">When the event was created, RFC 3339.</param>
/// <param name="Data">The event's data, as JSON.</param>
public sealed record UnknownWebhookEvent(string Id, string Type, string CreatedAt, JsonElement Data)
    : WebhookEvent(Id, Type, CreatedAt);
