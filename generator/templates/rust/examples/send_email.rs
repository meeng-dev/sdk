//! Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client at
//! another app, the address from MEENG_BASE_URL.
//!
//! ```sh
//! MEENG_API_KEY=mg_xxx cargo run --example send_email
//! ```

use meeng::{Client, EmailsSendParams};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = Client::from_env()?;
    let params = EmailsSendParams::new(
        vec!["ana@example.com".to_owned()],
        "Acme <contato@acme.com.br>",
    )
    .subject("Bem-vinda à Acme")
    .html("<p>Olá, Ana! Sua conta está pronta.</p>")
    .text("Olá, Ana! Sua conta está pronta.");

    // A key makes the send safe to repeat: the same key with the same body
    // answers the first result again, and nothing is sent twice.
    let sent = client
        .emails()
        .idempotency_key("boas-vindas-ana-2026-11-20")
        .send(&params)?;
    println!("e-mail na fila: {}", sent.email_id);
    Ok(())
}
