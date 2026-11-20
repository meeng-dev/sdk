using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;

namespace Meeng.Examples;

/// <summary>Sends up to 100 e-mails in one request.</summary>
public static class SendBatch
{
    public static async Task RunAsync()
    {
        var meeng = new MeengClient();
        var people = new Dictionary<string, string> { ["Ana"] = "ana@example.com", ["Bruno"] = "bruno@example.com", ["Carla"] = "carla@example.com" };
        var batch = people.Select(person => new EmailsBatchParams
        {
            From = "Acme <contato@acme.com.br>",
            To = [person.Value],
            Subject = "Seu extrato de novembro",
            Html = $"<p>Olá, {person.Key}! Seu extrato está disponível.</p>",
        }).ToList();

        var sent = await meeng.Emails.BatchAsync(batch, new RequestOptions { IdempotencyKey = "extrato-2026-11" });
        for (var i = 0; i < sent.Data.Count; i++)
        {
            Console.WriteLine($"{i + 1}: {sent.Data[i].EmailId}");
        }
    }
}
