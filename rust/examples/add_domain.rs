//! Adds a sending domain, prints the DNS records to publish and asks for them to
//! be checked.
//!
//! ```sh
//! MEENG_API_KEY=mg_xxx cargo run --example add_domain
//! ```

use meeng::{Client, DomainsCreateParams};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = Client::from_env()?;

    let domain = client
        .domains()
        .create(&DomainsCreateParams::new("acme.com.br", "us-east-1"))?;
    println!("publique estes registros no DNS de {}", domain.name);
    for record in &domain.dns_records {
        println!("  {:<5} {:<40} {}", record.type_, record.name, record.value);
    }

    let id = domain.id.to_string();
    client.domains().verify(&id)?;
    let checked = client.domains().get(&id)?;
    println!("situação: {}", checked.status);
    Ok(())
}
