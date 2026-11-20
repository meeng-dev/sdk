//! Sends up to 100 e-mails in one request.
//!
//! ```sh
//! MEENG_API_KEY=mg_xxx cargo run --example send_batch
//! ```

use meeng::{Client, EmailsBatchParams};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = Client::from_env()?;
    let people = [
        ("Ana", "ana@example.com"),
        ("Bruno", "bruno@example.com"),
        ("Carla", "carla@example.com"),
    ];
    let batch: Vec<EmailsBatchParams> = people
        .iter()
        .map(|(name, email)| {
            EmailsBatchParams::new(vec![(*email).to_owned()], "Acme <contato@acme.com.br>")
                .subject("Seu extrato de novembro")
                .html(format!("<p>Olá, {name}! Seu extrato está disponível.</p>"))
        })
        .collect();

    let sent = client
        .emails()
        .idempotency_key("extrato-2026-11")
        .batch(&batch)?;
    for (index, email) in sent.data.iter().enumerate() {
        println!("{}: {}", index + 1, email.email_id);
    }
    Ok(())
}
