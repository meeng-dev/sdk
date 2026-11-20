using System;
using System.Collections.Generic;
using System.Security.Cryptography;
using System.Text;
using System.Threading.Tasks;

namespace Meeng.Examples;

/// <summary>
/// Checks a webhook the way an endpoint does: the signature and the time are
/// verified against the raw body, and then the event is read as a typed value.
/// The request here is made in the program; a real endpoint reads the body and the
/// headers of the request it receives, and the secret from its settings.
/// </summary>
public static class VerifyWebhook
{
    public static Task RunAsync()
    {
        const string secret = "whsec_exemplo";
        const string body = """{"id":"call_1","type":"email.delivered","version":"2026-01-18","createdAt":"2026-11-20T12:00:00Z","teamId":7,"attempt":1,"data":{"id":"em_1","status":"DELIVERED","from":"contato@acme.com.br","to":["ana@example.com"],"occurredAt":"2026-11-20T12:00:00Z"}}""";

        var timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds().ToString();
        var signature = Convert.ToHexStringLower(HMACSHA256.HashData(Encoding.UTF8.GetBytes(secret), Encoding.UTF8.GetBytes($"{timestamp}.{body}")));
        var headers = new Dictionary<string, string> { ["X-Meeng-Signature"] = $"v1={signature}", ["X-Meeng-Timestamp"] = timestamp };

        try
        {
            var webhooks = new MeengClient("mg_exemplo").Webhooks(secret);
            switch (webhooks.ConstructEvent(body, headers))
            {
                case EmailWebhookEvent email:
                    Console.WriteLine($"e-mail {email.Data.Id} entregue a {string.Join(", ", email.Data.To)}");
                    break;
                case var other:
                    Console.WriteLine($"evento {other.Type}");
                    break;
            }
        }
        catch (WebhookVerificationException error)
        {
            Console.Error.WriteLine(error.Message);
            Environment.ExitCode = 1;
        }
        return Task.CompletedTask;
    }
}
