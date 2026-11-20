//! Checks a webhook the way an endpoint does: the signature and the time are
//! verified against the raw body, and then the event is read as a typed value.
//! The request here is made in the program; a real endpoint reads the body and the
//! headers of the request it receives, and the secret from its settings.
//!
//! ```sh
//! cargo run --example verify_webhook
//! ```

use std::time::{SystemTime, UNIX_EPOCH};

use hmac::{Hmac, KeyInit, Mac};
use meeng::{WebhookEvent, Webhooks};
use sha2::Sha256;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let secret = "whsec_exemplo";
    let body = r#"{"id":"call_1","type":"email.delivered","version":"2026-01-18","createdAt":"2026-11-20T12:00:00Z","teamId":7,"attempt":1,"data":{"id":"em_1","status":"DELIVERED","from":"contato@acme.com.br","to":["ana@example.com"],"occurredAt":"2026-11-20T12:00:00Z"}}"#;

    let timestamp = SystemTime::now()
        .duration_since(UNIX_EPOCH)?
        .as_millis()
        .to_string();
    let mut mac = Hmac::<Sha256>::new_from_slice(secret.as_bytes())?;
    mac.update(format!("{timestamp}.{body}").as_bytes());
    let signature: String = mac
        .finalize()
        .into_bytes()
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect();
    let headers = [
        ("X-Meeng-Signature", format!("v1={signature}")),
        ("X-Meeng-Timestamp", timestamp),
    ];

    match Webhooks::new(secret).construct_event(body.as_bytes(), headers)? {
        WebhookEvent::Email(event) => {
            println!(
                "e-mail {} entregue a {}",
                event.data.id,
                event.data.to.join(", ")
            );
        }
        other => println!("evento {other:?}"),
    }
    Ok(())
}
