using System;
using System.Collections.Generic;
using System.Linq;
using System.Net;
using System.Net.Http;
using System.Net.Sockets;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using Xunit;

namespace Meeng.Tests;

public class MeengTests
{
    private sealed record Seen(HttpMethod Method, Uri Url, Dictionary<string, string> Headers, string? Body);

    private sealed record Answer(HttpStatusCode Status, string? Json);

    /// <summary>Answers every request with what answer says, recording what was sent.</summary>
    private sealed class Recorder(Func<Seen, Answer> answer) : HttpMessageHandler
    {
        public List<Seen> Requests { get; } = [];

        protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            var headers = request.Headers.ToDictionary(h => h.Key, h => string.Join(",", h.Value), StringComparer.OrdinalIgnoreCase);
            string? body = null;
            if (request.Content is not null)
            {
                foreach (var header in request.Content.Headers)
                {
                    headers[header.Key] = string.Join(",", header.Value);
                }
                body = await request.Content.ReadAsStringAsync(cancellationToken);
            }
            var seen = new Seen(request.Method, request.RequestUri!, headers, body);
            Requests.Add(seen);
            var reply = answer(seen);
            var response = new HttpResponseMessage(reply.Status) { Content = new StringContent(reply.Json ?? "", Encoding.UTF8, "application/json") };
            response.Headers.TryAddWithoutValidation("Retry-After", "7");
            return response;
        }
    }

    private static MeengClient Client(Recorder recorder) =>
        new("mg_test_key", new MeengClientOptions { BaseUrl = "https://meeng.test/", Handler = recorder });

    private static Recorder Replying(HttpStatusCode status, string json) => new(_ => new Answer(status, json));

    [Fact]
    public async Task SendingAnEmailCarriesTheKeyTheBodyAndTheIdempotencyKey()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"emailId":"em_1"}""");
        var sent = await Client(recorder).Emails.SendAsync(
            new EmailsSendParams { From = "hello@acme.com", To = ["a@x.com", "b@x.com"], Subject = "Hi", Html = "<p>Hi</p>" },
            new RequestOptions { IdempotencyKey = "order-1" });
        Assert.Equal("em_1", sent.EmailId);
        var seen = Assert.Single(recorder.Requests);
        Assert.Equal(HttpMethod.Post, seen.Method);
        Assert.Equal("https://meeng.test/api/v1/emails", seen.Url.ToString());
        Assert.Equal("Bearer mg_test_key", seen.Headers["Authorization"]);
        Assert.Equal("order-1", seen.Headers["Idempotency-Key"]);
        Assert.Equal("application/json", seen.Headers["Content-Type"]);
        Assert.Equal("meeng-csharp/" + MeengClient.Version, seen.Headers["User-Agent"]);
        using var body = JsonDocument.Parse(seen.Body!);
        Assert.Equal("hello@acme.com", body.RootElement.GetProperty("from").GetString());
        Assert.Equal(["a@x.com", "b@x.com"], body.RootElement.GetProperty("to").EnumerateArray().Select(e => e.GetString()!).ToArray());
        Assert.False(body.RootElement.TryGetProperty("cc", out _));
    }

    [Fact]
    public async Task AWriteWithNoBodyOfItsOwnSendsAnEmptyObject()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"emailId":"em_1"}""");
        await Client(recorder).Emails.CancelAsync("em_1");
        var seen = Assert.Single(recorder.Requests);
        Assert.Equal(HttpMethod.Post, seen.Method);
        Assert.Equal("/api/v1/emails/em_1/cancel", seen.Url.AbsolutePath);
        Assert.Equal("{}", seen.Body);
    }

    [Fact]
    public async Task ASendWithoutAnIdempotencyKeySendsNone()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"emailId":"em_1"}""");
        await Client(recorder).Emails.SendAsync(new EmailsSendParams { From = "hello@acme.com", To = ["a@x.com"], Text = "Hi" });
        Assert.False(recorder.Requests[0].Headers.ContainsKey("Idempotency-Key"));
    }

    [Fact]
    public async Task ABatchAnswersTheListOfIds()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"data":[{"emailId":"em_1"},{"emailId":"em_2"}]}""");
        var got = await Client(recorder).Emails.BatchAsync(
        [
            new EmailsBatchParams { From = "hello@acme.com", To = ["a@x.com"], Subject = "One", Text = "1" },
            new EmailsBatchParams { From = "hello@acme.com", To = ["b@x.com"], Subject = "Two", Text = "2" },
        ]);
        Assert.Equal("em_2", got.Data[1].EmailId);
        var seen = Assert.Single(recorder.Requests);
        Assert.Equal("/api/v1/emails/batch", seen.Url.AbsolutePath);
        using var body = JsonDocument.Parse(seen.Body!);
        Assert.Equal(2, body.RootElement.GetArrayLength());
    }

    [Fact]
    public async Task ListingEmailsSendsTheFiltersAListOnceForEachValue()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"data":[{"id":"em_1","to":["a@x.com"],"replyTo":[],"cc":[],"bcc":[],"from":"h@acme.com","subject":"Hi","html":null,"text":"Hi","createdAt":"x","updatedAt":"x","latestStatus":"DELIVERED","scheduledAt":null,"domainId":3}],"count":1}""");
        var got = await Client(recorder).Emails.ListAsync(new EmailsListQuery { Page = 2, Limit = 10, StartDate = "2026-11-01T00:00:00Z", DomainId = ["1", "2"] });
        Assert.Equal(1, got.Count);
        Assert.Equal(["a@x.com"], got.Data[0].To);
        Assert.Equal("DELIVERED", got.Data[0].LatestStatus);
        var seen = Assert.Single(recorder.Requests);
        Assert.Equal(HttpMethod.Get, seen.Method);
        Assert.Equal("https://meeng.test/api/v1/emails?page=2&limit=10&startDate=2026-11-01T00%3A00%3A00Z&domainId=1&domainId=2", seen.Url.ToString());
    }

    [Fact]
    public async Task APathParameterIsEscaped()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"id":"a/b c","teamId":1,"to":["a@x.com"],"from":"h@acme.com","subject":"Hi","html":null,"text":null,"createdAt":"x","updatedAt":"x","emailEvents":[{"emailId":"a/b c","status":"SENT","createdAt":"x","data":{"ip":"1.2.3.4"}}]}""");
        var email = await Client(recorder).Emails.GetAsync("a/b c");
        Assert.Equal("https://meeng.test/api/v1/emails/a%2Fb%20c", recorder.Requests[0].Url.AbsoluteUri);
        Assert.Equal("SENT", email.EmailEvents[0].Status);
    }

    [Fact]
    public async Task AListTheApiAnswersWithoutAnObjectAroundItIsAList()
    {
        var recorder = Replying(HttpStatusCode.OK, """[{"id":"cb_1","name":"News","teamId":1,"properties":{"plan":"string"},"variables":["plan"],"emoji":"x","doubleOptInEnabled":false,"doubleOptInFrom":null,"doubleOptInSubject":null,"doubleOptInContent":null,"createdAt":"x","updatedAt":"x","_count":{"contacts":4}}]""");
        var books = await Client(recorder).ContactBooks.ListAsync();
        Assert.Equal("News", Assert.Single(books).Name);
        Assert.Equal(4, books[0].Count!.Contacts);
    }

    [Fact]
    public async Task ADeleteCanCarryABody()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"success":true,"count":2}""");
        var got = await Client(recorder).Contacts.BulkDeleteAsync("cb_1", new ContactsBulkDeleteParams { ContactIds = ["c1", "c2"] });
        Assert.Equal(2, got.Count);
        var seen = Assert.Single(recorder.Requests);
        Assert.Equal(HttpMethod.Delete, seen.Method);
        Assert.Equal("/api/v1/contactBooks/cb_1/contacts/bulk", seen.Url.AbsolutePath);
        Assert.Equal("""{"contactIds":["c1","c2"]}""", seen.Body);
    }

    [Fact]
    public async Task UpsertTakesTheContactsIdInThePath()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"contactId":"c_1"}""");
        var got = await Client(recorder).Contacts.UpsertAsync("cb_1", "c_1", new ContactsUpsertParams { Email = "a@x.com", FirstName = "Ana" });
        Assert.Equal("c_1", got.ContactId);
        Assert.Equal(HttpMethod.Put, recorder.Requests[0].Method);
        Assert.Equal("/api/v1/contactBooks/cb_1/contacts/c_1", recorder.Requests[0].Url.AbsolutePath);
    }

    [Theory]
    [InlineData(409, """{"error":{"code":"NOT_UNIQUE","message":"Idempotency-Key already used"}}""", "NOT_UNIQUE", "Idempotency-Key already used")]
    [InlineData(403, """{"error":"Forbidden"}""", "INTERNAL_SERVER_ERROR", "Forbidden")]
    [InlineData(400, """{"code":"BAD_REQUEST","message":"bad"}""", "BAD_REQUEST", "bad")]
    [InlineData(400, """{"success":false,"error":{"issues":[{"code":"invalid_type","path":["from"],"message":"Required"}],"name":"ZodError"}}""", "ZodError", "from: Required")]
    [InlineData(502, "<html>", "INTERNAL_SERVER_ERROR", "Bad Gateway")]
    public async Task ARefusalThrowsInEveryShapeTheApiWritesIt(int status, string body, string code, string message)
    {
        var refusal = await Assert.ThrowsAsync<MeengException>(() => Client(Replying((HttpStatusCode)status, body)).Emails.CancelAsync("em_1"));
        Assert.Equal(status, refusal.Status);
        Assert.Equal(code, refusal.Code);
        Assert.Equal(7, refusal.RetryAfter);
        Assert.Equal($"meeng: {status} {code}: {message}", refusal.Message);
    }

    [Fact]
    public async Task AnEmailsRecipientsAreAList()
    {
        var recorder = Replying(HttpStatusCode.OK, """{"id":"em_1","teamId":1,"to":["a@x.com","b@x.com"],"from":"h@acme.com","subject":"Hi","html":null,"text":null,"createdAt":"x","updatedAt":"x","emailEvents":[]}""");
        var email = await Client(recorder).Emails.GetAsync("em_1");
        Assert.Equal(["a@x.com", "b@x.com"], email.To);
    }

    [Fact]
    public void TheKeyAndTheAddressComeFromTheEnvironment()
    {
        var saved = (Environment.GetEnvironmentVariable("MEENG_API_KEY"), Environment.GetEnvironmentVariable("MEENG_BASE_URL"));
        try
        {
            Environment.SetEnvironmentVariable("MEENG_API_KEY", "mg_env");
            Environment.SetEnvironmentVariable("MEENG_BASE_URL", "https://self.hosted.test");
            Assert.Equal("https://self.hosted.test/api", new MeengClient().Url);
            Environment.SetEnvironmentVariable("MEENG_API_KEY", null);
            Assert.Throws<ArgumentException>(() => new MeengClient());
        }
        finally
        {
            Environment.SetEnvironmentVariable("MEENG_API_KEY", saved.Item1);
            Environment.SetEnvironmentVariable("MEENG_BASE_URL", saved.Item2);
        }
    }

    [Fact]
    public void TheAddressMustBeHttpsExceptToLocalhost()
    {
        foreach (var bad in new[] { "http://app.meeng.test", "ftp://app.meeng.test", "app.meeng.test", "http://localhost.evil.test" })
        {
            Assert.Throws<ArgumentException>(() => new MeengClient("k", new MeengClientOptions { BaseUrl = bad }));
        }
        foreach (var good in new[] { "https://app.meeng.test", "http://localhost:3000", "http://127.0.0.1:3000", "http://[::1]:3000" })
        {
            _ = new MeengClient("k", new MeengClientOptions { BaseUrl = good });
        }
    }

    [Fact]
    public async Task TheDefaultHttpClientNeverFollowsARedirect()
    {
        using var listener = new TcpListener(IPAddress.Loopback, 0);
        listener.Start();
        var port = ((IPEndPoint)listener.LocalEndpoint).Port;
        var serving = Task.Run(async () =>
        {
            using var socket = await listener.AcceptTcpClientAsync();
            var stream = socket.GetStream();
            var buffer = new byte[4096];
            _ = await stream.ReadAsync(buffer);
            await stream.WriteAsync("HTTP/1.1 302 Found\r\nLocation: https://elsewhere.test/\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"u8.ToArray());
        });
        var client = new MeengClient("k", new MeengClientOptions { BaseUrl = $"http://127.0.0.1:{port}" });
        var refusal = await Assert.ThrowsAsync<MeengException>(() => client.Emails.CancelAsync("em_1"));
        Assert.Equal(302, refusal.Status);
        await serving;
    }

    private const string Secret = "whsec_test";
    private const string Body = """{"id":"evt_1","type":"email.delivered","version":"2026-01-18","createdAt":"2026-11-20T12:00:00Z","teamId":7,"attempt":2,"data":{"id":"em_1","status":"DELIVERED","from":"h@acme.com","to":["a@x.com"],"occurredAt":"2026-11-20T12:00:00Z"}}""";

    private static string Sign(string timestamp, string text = Body, string secret = Secret) =>
        "v1=" + Convert.ToHexString(HMACSHA256.HashData(Encoding.UTF8.GetBytes(secret), Encoding.UTF8.GetBytes($"{timestamp}.{text}"))).ToLowerInvariant();

    [Fact]
    public void ASignedWebhookIsVerifiedAndReadAsATypedEvent()
    {
        var timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds().ToString();
        var webhooks = new MeengClient("k").Webhooks(Secret);
        var headers = new Dictionary<string, string> { ["X-Meeng-Signature"] = Sign(timestamp), ["X-Meeng-Timestamp"] = timestamp };
        Assert.True(webhooks.Verify(Body, headers));
        var delivered = Assert.IsType<EmailWebhookEvent>(webhooks.ConstructEvent(Body, headers));
        Assert.Equal("email.delivered", delivered.Type);
        Assert.Equal(("2026-01-18", 7L, 2), (delivered.Version, delivered.TeamId, delivered.Attempt));
        Assert.Equal(["a@x.com"], delivered.Data.To);
        Assert.Equal(EmailBasePayload.StatusDelivered, delivered.Data.Status);
    }

    [Fact]
    public void TheUseSendHeadersAreAcceptedAsWellInAnyCase()
    {
        var timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds().ToString();
        var headers = new Dictionary<string, string> { ["x-usesend-signature"] = Sign(timestamp), ["X-UseSend-Timestamp"] = timestamp };
        Assert.True(new MeengClient("k").Webhooks(Secret).Verify(Body, headers));
    }

    [Fact]
    public void AWebhookIsRefusedWithTheCodeThatSaysWhy()
    {
        var now = DateTimeOffset.FromUnixTimeMilliseconds(1_800_000_000_000);
        var timestamp = now.ToUnixTimeMilliseconds().ToString();
        var old = (now.ToUnixTimeMilliseconds() - 6 * 60 * 1000).ToString();
        var webhooks = new MeengClient("k").Webhooks(Secret);
        var good = new Dictionary<string, string> { ["X-Meeng-Signature"] = Sign(timestamp), ["X-Meeng-Timestamp"] = timestamp };
        var stale = new Dictionary<string, string> { ["X-Meeng-Signature"] = Sign(old), ["X-Meeng-Timestamp"] = old };

        string Refused(Dictionary<string, string> headers, string body = Body, WebhookOptions? options = null)
        {
            try
            {
                webhooks.ConstructEvent(body, headers, (options ?? new WebhookOptions()) with { Now = now });
                return "ACCEPTED";
            }
            catch (WebhookVerificationException e)
            {
                return e.Code;
            }
        }

        Assert.Equal("ACCEPTED", Refused(good));
        Assert.Equal("MISSING_SIGNATURE", Refused(new() { ["X-Meeng-Timestamp"] = timestamp }));
        Assert.Equal("MISSING_TIMESTAMP", Refused(new() { ["X-Meeng-Signature"] = Sign(timestamp) }));
        Assert.Equal("INVALID_SIGNATURE_FORMAT", Refused(new() { ["X-Meeng-Signature"] = Sign(timestamp)[3..], ["X-Meeng-Timestamp"] = timestamp }));
        Assert.Equal("INVALID_TIMESTAMP", Refused(new() { ["X-Meeng-Signature"] = Sign(timestamp), ["X-Meeng-Timestamp"] = "yesterday" }));
        Assert.Equal("TIMESTAMP_OUT_OF_RANGE", Refused(stale));
        Assert.Equal("ACCEPTED", Refused(stale, options: new WebhookOptions { Tolerance = TimeSpan.FromMilliseconds(-1) }));
        Assert.Equal("ACCEPTED", Refused(stale, options: new WebhookOptions { Tolerance = TimeSpan.FromMinutes(10) }));
        Assert.Equal("SIGNATURE_MISMATCH", Refused(good, options: new WebhookOptions { Secret = "another" }));
        Assert.Equal("SIGNATURE_MISMATCH", Refused(good, Body.Replace("em_1", "em_2")));
        Assert.Equal("INVALID_JSON", Refused(new() { ["X-Meeng-Signature"] = Sign(timestamp, "not json"), ["X-Meeng-Timestamp"] = timestamp }, "not json"));
    }
}
