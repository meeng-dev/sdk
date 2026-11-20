import Meeng

/// Adds a sending domain, prints the DNS records to publish and asks for them to be checked.
func addDomain() async throws {
    let meeng = MeengClient()
    let domain = try await meeng.domains.create(DomainsCreateParams(name: "acme.com.br", region: "us-east-1"))
    print("publique estes registros no DNS de \(domain.name)")
    for record in domain.dnsRecords {
        print("  \(record.type.padding(toLength: 5, withPad: " ", startingAt: 0)) \(record.name.padding(toLength: 40, withPad: " ", startingAt: 0)) \(record.value)")
    }

    _ = try await meeng.domains.verify(String(domain.id))
    let checked = try await meeng.domains.get(String(domain.id))
    print("situação: \(checked.status)")
}
