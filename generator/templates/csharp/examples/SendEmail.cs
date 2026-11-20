using System;
using System.Threading.Tasks;

namespace Meeng.Examples;

/// <summary>Sends one e-mail.</summary>
public static class SendEmail
{
    public static async Task RunAsync()
    {
        var meeng = new MeengClient();
        var sent = await meeng.Emails.SendAsync(
            new EmailsSendParams
            {
                From = "Acme <contato@acme.com.br>",
                To = ["ana@example.com"],
                Subject = "Bem-vinda à Acme",
                Html = "<p>Olá, Ana! Sua conta está pronta.</p>",
                Text = "Olá, Ana! Sua conta está pronta.",
            },
            // A key makes the send safe to repeat: the same key with the same body
            // answers the first result again, and nothing is sent twice.
            new RequestOptions { IdempotencyKey = "boas-vindas-ana-2026-11-20" });
        Console.WriteLine($"e-mail na fila: {sent.EmailId}");
    }
}
