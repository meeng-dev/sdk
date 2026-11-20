//! Schedules an e-mail, moves it to another time and cancels it.
//!
//! ```sh
//! MEENG_API_KEY=mg_xxx cargo run --example schedule_email
//! ```

use std::time::{Duration, SystemTime, UNIX_EPOCH};

use meeng::{Client, EmailsSendParams, EmailsUpdateParams};

/// An RFC 3339 time in UTC, `hours` from now.
fn in_hours(hours: u64) -> String {
    let seconds = SystemTime::now()
        .checked_add(Duration::from_secs(hours * 3600))
        .and_then(|time| time.duration_since(UNIX_EPOCH).ok())
        .map_or(0, |elapsed| elapsed.as_secs());
    let (days, rest) = (seconds / 86_400, seconds % 86_400);
    // Civil date from the day count (Howard Hinnant's algorithm).
    let z = days as i64 + 719_468;
    let era = z.div_euclid(146_097);
    let doe = z.rem_euclid(146_097);
    let yoe = (doe - doe / 1_460 + doe / 36_524 - doe / 146_096) / 365;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let day = doy - (153 * mp + 2) / 5 + 1;
    let month = if mp < 10 { mp + 3 } else { mp - 9 };
    let year = yoe + era * 400 + i64::from(month <= 2);
    format!(
        "{year:04}-{month:02}-{day:02}T{:02}:{:02}:{:02}Z",
        rest / 3600,
        rest % 3600 / 60,
        rest % 60
    )
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = Client::from_env()?;

    let params = EmailsSendParams::new(
        vec!["ana@example.com".to_owned()],
        "Acme <contato@acme.com.br>",
    )
    .subject("Lembrete: sua reunião é amanhã")
    .html("<p>Nos vemos às 10h.</p>")
    .scheduled_at(in_hours(24));
    let sent = client.emails().send(&params)?;
    println!("agendado: {}", sent.email_id);

    client
        .emails()
        .update(&sent.email_id, &EmailsUpdateParams::new(in_hours(48)))?;
    println!("remarcado");

    client.emails().cancel(&sent.email_id)?;
    println!("cancelado");
    Ok(())
}
