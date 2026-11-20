using System;
using System.Net.Http;

namespace Meeng;

/// <summary>How a <see cref="MeengClient"/> reaches the API.</summary>
public sealed class MeengClientOptions
{
    /// <summary>
    /// Another address, such as a local server; <see cref="MeengClient.DefaultBaseUrl"/> when null.
    /// It must be https; http is accepted only for localhost, 127.0.0.0/8 and ::1, and any other
    /// makes the <see cref="MeengClient"/> constructor throw <see cref="ArgumentException"/>.
    /// </summary>
    public string? BaseUrl { get; set; }

    /// <summary>How long a call waits for its response; 30 seconds when null.</summary>
    public TimeSpan? Timeout { get; set; }

    /// <summary>
    /// How many times a call that may be repeated is repeated after the first attempt; 2 by
    /// default, and 0 sends every call once. See <see cref="MeengClient"/> for which calls are repeated.
    /// </summary>
    public int MaxRetries { get; set; } = 2;

    /// <summary>
    /// The wait before the first repetition; 500 ms when null. Each next repetition waits twice as
    /// long, and a Retry-After the API answers replaces the wait.
    /// </summary>
    public TimeSpan? RetryDelay { get; set; }

    /// <summary>
    /// The HttpClient that sends the requests, such as one from
    /// <c>IHttpClientFactory</c>; the SDK's shared one when null. Its own
    /// timeout applies on top of <see cref="Timeout"/>.
    /// </summary>
    public HttpClient? HttpClient { get; set; }

    /// <summary>A handler the requests are sent through, when <see cref="HttpClient"/> is null: a test's, or a proxy's.</summary>
    public HttpMessageHandler? Handler { get; set; }
}
