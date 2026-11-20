using System;
using System.Threading.Tasks;

namespace Meeng.Examples;

/// <summary>Schedules an e-mail, moves it to another time and cancels it.</summary>
public static class ScheduleEmail
{
    private static string InHours(int hours) => DateTime.UtcNow.AddHours(hours).ToString("yyyy-MM-dd'T'HH:mm:ss'Z'");

    public static async Task RunAsync()
    {
        var meeng = new MeengClient();
        var sent = await meeng.Emails.SendAsync(new EmailsSendParams
        {
            From = "Acme <contato@acme.com.br>",
            To = ["ana@example.com"],
            Subject = "Lembrete: sua reunião é amanhã",
            Html = "<p>Nos vemos às 10h.</p>",
            ScheduledAt = InHours(24),
        });
        Console.WriteLine($"agendado: {sent.EmailId}");

        await meeng.Emails.UpdateAsync(sent.EmailId, new EmailsUpdateParams { ScheduledAt = InHours(48) });
        Console.WriteLine("remarcado");

        await meeng.Emails.CancelAsync(sent.EmailId);
        Console.WriteLine("cancelado");
    }
}
