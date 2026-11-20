using System.Text.Json;

namespace Meeng;

/// <summary>
/// A webhook meeng sent, verified: one record a kind of data
/// (<see cref="EmailWebhookEvent"/> carries an <see cref="EmailBasePayload"/>…), or
/// <see cref="UnknownWebhookEvent"/> for a type this SDK does not know yet.
/// Within a record, <see cref="Type"/> says what happened.
/// </summary>
/// <param name="Id">The event's id.</param>
/// <param name="Type">What happened: <c>email.delivered</c>…</param>
/// <param name="CreatedAt">When the event was created, RFC 3339.</param>
public abstract record WebhookEvent(string Id, string Type, string CreatedAt);

/// <summary>An event of a type this SDK does not know yet, its data as it was sent.</summary>
/// <param name="Id">The event's id.</param>
/// <param name="Type">What happened.</param>
/// <param name="CreatedAt">When the event was created, RFC 3339.</param>
/// <param name="Data">The event's data, as JSON.</param>
public sealed record UnknownWebhookEvent(string Id, string Type, string CreatedAt, JsonElement Data)
    : WebhookEvent(Id, Type, CreatedAt);
