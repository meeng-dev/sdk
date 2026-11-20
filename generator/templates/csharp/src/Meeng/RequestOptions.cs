namespace Meeng;

/// <summary>Options for one send.</summary>
public sealed record RequestOptions
{
    /// <summary>
    /// Makes a send safe to retry: the same key with the same body answers the
    /// first result again; with another body, a 409 <c>NOT_UNIQUE</c>. The key
    /// is up to 256 characters and expires after 24 hours.
    /// </summary>
    public string? IdempotencyKey { get; init; }
}
