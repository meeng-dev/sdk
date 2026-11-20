// The examples of the C# SDK, one program that runs the example it is asked for:
//
//   MEENG_API_KEY=mg_xxx dotnet run --project examples -- send-email
//
// send-email, send-batch, schedule-email, add-domain or verify-webhook. The key
// comes from MEENG_API_KEY and, to point the client at another app, the address
// from MEENG_BASE_URL.
using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Meeng;
using Meeng.Examples;

var examples = new Dictionary<string, Func<Task>>
{
    ["send-email"] = SendEmail.RunAsync,
    ["send-batch"] = SendBatch.RunAsync,
    ["schedule-email"] = ScheduleEmail.RunAsync,
    ["add-domain"] = AddDomain.RunAsync,
    ["verify-webhook"] = VerifyWebhook.RunAsync,
};

if (args.Length != 1 || !examples.TryGetValue(args[0], out var run))
{
    Console.Error.WriteLine("usage: Meeng.Examples " + string.Join(" | ", examples.Keys));
    return 2;
}

try
{
    await run();
    return 0;
}
catch (MeengException error)
{
    Console.Error.WriteLine(error.Message);
    return 1;
}
