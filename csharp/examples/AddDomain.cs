using System;
using System.Globalization;
using System.Threading.Tasks;

namespace Meeng.Examples;

/// <summary>Adds a sending domain, prints the DNS records to publish and asks for them to be checked.</summary>
public static class AddDomain
{
    public static async Task RunAsync()
    {
        var meeng = new MeengClient();
        var domain = await meeng.Domains.CreateAsync(new DomainsCreateParams { Name = "acme.com.br", Region = "us-east-1" });
        Console.WriteLine($"publique estes registros no DNS de {domain.Name}");
        foreach (var record in domain.DnsRecords)
        {
            Console.WriteLine($"  {record.Type,-5} {record.Name,-40} {record.Value}");
        }

        var id = domain.Id.ToString(CultureInfo.InvariantCulture);
        await meeng.Domains.VerifyAsync(id);
        var checkedDomain = await meeng.Domains.GetAsync(id);
        Console.WriteLine($"situação: {checkedDomain.Status}");
    }
}
