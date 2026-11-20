use std::io::{BufRead, BufReader, Read, Write};
use std::net::TcpListener;
use std::sync::{Arc, Mutex};
use std::thread;

use hmac::{Hmac, KeyInit, Mac};
use meeng::{
    BoxError, Client, EmailsBatchParams, EmailsListQuery, EmailsSendParams, Error, HttpRequest,
    HttpResponse, WebhookEvent, WebhookOptions, Webhooks,
};
use sha2::Sha256;

type Seen = Arc<Mutex<Vec<HttpRequest>>>;

fn client(answer: impl Fn(&HttpRequest) -> HttpResponse + Send + Sync + 'static) -> (Client, Seen) {
    let seen: Seen = Arc::default();
    let record = Arc::clone(&seen);
    let transport = move |request: HttpRequest| -> Result<HttpResponse, BoxError> {
        let response = answer(&request);
        record.lock().unwrap().push(request);
        Ok(response)
    };
    let client = Client::builder("mg_test_key")
        .base_url("https://meeng.test/")
        .transport(transport)
        .build();
    (client, seen)
}

fn json(status: u16, body: serde_json::Value) -> HttpResponse {
    HttpResponse::new(status, body.to_string())
        .with_header("Content-Type", "application/json")
        .with_header("Retry-After", "7")
}

fn body_of(request: &HttpRequest) -> serde_json::Value {
    serde_json::from_slice(request.body.as_deref().unwrap()).unwrap()
}

#[test]
fn sending_an_email_carries_the_key_the_body_and_the_idempotency_key() {
    let (client, seen) = client(|_| json(200, serde_json::json!({"emailId": "em_1"})));
    let params = EmailsSendParams::new(
        vec!["a@x.com".to_owned(), "b@x.com".to_owned()],
        "hello@acme.com",
    )
    .subject("Hi")
    .html("<p>Hi</p>");
    let sent = client
        .emails()
        .idempotency_key("order-1")
        .send(&params)
        .unwrap();
    assert_eq!(sent.email_id.as_deref(), Some("em_1"));
    let seen = seen.lock().unwrap();
    assert_eq!(seen[0].method, "POST");
    assert_eq!(seen[0].url, "https://meeng.test/api/v1/emails");
    assert_eq!(seen[0].header("authorization"), Some("Bearer mg_test_key"));
    assert_eq!(seen[0].header("idempotency-key"), Some("order-1"));
    assert_eq!(seen[0].header("content-type"), Some("application/json"));
    assert_eq!(
        seen[0].header("user-agent"),
        Some(format!("meeng-rust/{}", meeng::VERSION).as_str())
    );
    assert_eq!(
        body_of(&seen[0]),
        serde_json::json!({"from": "hello@acme.com", "to": ["a@x.com", "b@x.com"], "subject": "Hi", "html": "<p>Hi</p>"})
    );
}

#[test]
fn a_write_with_no_body_of_its_own_sends_an_empty_object() {
    let (client, seen) = client(|_| json(200, serde_json::json!({"emailId": "em_1"})));
    client.emails().cancel("em_1").unwrap();
    let seen = seen.lock().unwrap();
    assert_eq!(seen[0].method, "POST");
    assert_eq!(seen[0].url, "https://meeng.test/api/v1/emails/em_1/cancel");
    assert_eq!(seen[0].body.as_deref(), Some(b"{}".as_slice()));
}

#[test]
fn a_send_without_an_idempotency_key_sends_none() {
    let (client, seen) = client(|_| json(200, serde_json::json!({"emailId": "em_1"})));
    client
        .emails()
        .send(&EmailsSendParams::new(vec!["a@x.com".to_owned()], "hello@acme.com").text("Hi"))
        .unwrap();
    assert_eq!(seen.lock().unwrap()[0].header("idempotency-key"), None);
}

#[test]
fn a_batch_answers_the_list_of_ids() {
    let (client, seen) = client(|_| {
        json(
            200,
            serde_json::json!({"data": [{"emailId": "em_1"}, {"emailId": "em_2"}]}),
        )
    });
    let got = client
        .emails()
        .batch(&[
            EmailsBatchParams::new(vec!["a@x.com".to_owned()], "hello@acme.com")
                .subject("One")
                .text("1"),
            EmailsBatchParams::new(vec!["b@x.com".to_owned()], "hello@acme.com")
                .subject("Two")
                .text("2"),
        ])
        .unwrap();
    assert_eq!(got.data[1].email_id, "em_2");
    let seen = seen.lock().unwrap();
    assert_eq!(seen[0].url, "https://meeng.test/api/v1/emails/batch");
    assert_eq!(body_of(&seen[0]).as_array().unwrap().len(), 2);
}

#[test]
fn listing_emails_sends_the_filters_a_list_once_for_each_value() {
    let (client, seen) = client(|_| {
        json(
            200,
            serde_json::json!({"data": [{"id": "em_1", "to": "a@x.com", "from": "h@acme.com", "subject": "Hi", "html": null, "text": "Hi", "createdAt": "x", "updatedAt": "x", "latestStatus": "DELIVERED", "scheduledAt": null, "domainId": 3}], "count": 1}),
        )
    });
    let query = EmailsListQuery::new()
        .page(2)
        .limit(10)
        .start_date("2026-11-01T00:00:00Z")
        .domain_id(["1", "2"]);
    let got = client.emails().list(&query).unwrap();
    assert_eq!(got.count, 1);
    assert_eq!(got.data[0].to, vec!["a@x.com".to_owned()]);
    assert_eq!(
        got.data[0].latest_status.as_ref().map(|s| s.as_str()),
        Some("DELIVERED")
    );
    assert_eq!(
        seen.lock().unwrap()[0].url,
        "https://meeng.test/api/v1/emails?page=2&limit=10&startDate=2026-11-01T00%3A00%3A00Z&domainId=1&domainId=2"
    );
}

#[test]
fn a_path_parameter_is_escaped() {
    let (client, seen) = client(|_| {
        json(
            200,
            serde_json::json!({"id": "a/b c", "teamId": 1, "to": ["a@x.com"], "from": "h@acme.com", "subject": "Hi", "html": null, "text": null, "createdAt": "x", "updatedAt": "x", "emailEvents": [{"emailId": "a/b c", "status": "SENT", "createdAt": "x", "data": {"ip": "1.2.3.4"}}]}),
        )
    });
    let email = client.emails().get("a/b c").unwrap();
    assert_eq!(
        seen.lock().unwrap()[0].url,
        "https://meeng.test/api/v1/emails/a%2Fb%20c"
    );
    assert_eq!(email.email_events[0].status.as_str(), "SENT");
}

#[test]
fn a_list_the_api_answers_without_an_object_around_it_is_a_list() {
    let (client, _) = client(|_| {
        json(
            200,
            serde_json::json!([{"id": "cb_1", "name": "News", "teamId": 1, "properties": {"plan": "string"}, "variables": ["plan"], "emoji": "x", "createdAt": "x", "updatedAt": "x", "_count": {"contacts": 4}}]),
        )
    });
    let books = client.contact_books().list().unwrap();
    assert_eq!(books[0].name, "News");
    assert_eq!(books[0].count.as_ref().unwrap().contacts, 4);
}

#[test]
fn a_delete_can_carry_a_body() {
    let (client, seen) = client(|_| json(200, serde_json::json!({"success": true, "count": 2})));
    let got = client
        .contacts()
        .bulk_delete(
            "cb_1",
            &meeng::ContactsBulkDeleteParams::new(vec!["c1".to_owned(), "c2".to_owned()]),
        )
        .unwrap();
    assert_eq!(got.count, 2);
    let seen = seen.lock().unwrap();
    assert_eq!(seen[0].method, "DELETE");
    assert_eq!(
        seen[0].url,
        "https://meeng.test/api/v1/contactBooks/cb_1/contacts/bulk"
    );
    assert_eq!(
        body_of(&seen[0]),
        serde_json::json!({"contactIds": ["c1", "c2"]})
    );
}

#[test]
fn upsert_takes_the_contacts_id_in_the_path() {
    let (client, seen) = client(|_| json(200, serde_json::json!({"contactId": "c_1"})));
    let got = client
        .contacts()
        .upsert(
            "cb_1",
            "c_1",
            &meeng::ContactsUpsertParams::new("a@x.com").first_name("Ana"),
        )
        .unwrap();
    assert_eq!(got.contact_id, "c_1");
    let seen = seen.lock().unwrap();
    assert_eq!(seen[0].method, "PUT");
    assert_eq!(
        seen[0].url,
        "https://meeng.test/api/v1/contactBooks/cb_1/contacts/c_1"
    );
}

#[test]
fn a_refusal_is_an_error_in_every_shape_the_api_writes_it() {
    let cases = [
        (409, serde_json::json!({"error": {"code": "NOT_UNIQUE", "message": "Idempotency-Key already used"}}).to_string(), "NOT_UNIQUE", "Idempotency-Key already used"),
        (403, serde_json::json!({"error": "Forbidden"}).to_string(), "INTERNAL_SERVER_ERROR", "Forbidden"),
        (400, serde_json::json!({"code": "BAD_REQUEST", "message": "bad"}).to_string(), "BAD_REQUEST", "bad"),
        (502, "<html>".to_owned(), "INTERNAL_SERVER_ERROR", "Bad Gateway"),
    ];
    for (status, body, code, message) in cases {
        let (client, _) = client(move |_| {
            HttpResponse::new(status, body.clone()).with_header("Retry-After", "7")
        });
        let error = client.emails().cancel("em_1").unwrap_err();
        let refusal = error.api().expect("an API error");
        assert_eq!(refusal.status, status);
        assert_eq!(refusal.code, code);
        assert_eq!(refusal.retry_after, Some(std::time::Duration::from_secs(7)));
        assert_eq!(
            error.to_string(),
            format!("meeng: {status} {code}: {message}")
        );
    }
}

#[test]
fn an_emails_recipients_may_come_as_one_string_or_a_list() {
    let (client, _) = client(|_| {
        json(
            200,
            serde_json::json!({"id": "em_1", "teamId": 1, "to": "a@x.com", "from": "h@acme.com", "subject": "Hi", "html": null, "text": null, "createdAt": "x", "updatedAt": "x", "emailEvents": []}),
        )
    });
    assert_eq!(
        client.emails().get("em_1").unwrap().to,
        vec!["a@x.com".to_owned()]
    );
}

#[test]
fn the_address_must_be_https_except_to_localhost() {
    for base in [
        "http://app.meeng.test",
        "ftp://app.meeng.test",
        "app.meeng.test",
        "http://localhost.evil.test",
    ] {
        let client = Client::builder("k")
            .base_url(base)
            .transport(|_: HttpRequest| -> Result<HttpResponse, BoxError> { panic!("sent") })
            .build();
        assert!(
            matches!(client.emails().get("em_1"), Err(Error::InsecureBaseUrl)),
            "{base}"
        );
    }
    assert_eq!(
        Client::builder("k")
            .base_url("https://app.meeng.test/")
            .build()
            .url(),
        "https://app.meeng.test/api"
    );
}

#[test]
fn the_standard_http_transport_sends_a_body_on_a_delete_and_never_follows_a_redirect() {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();
    let port = listener.local_addr().unwrap().port();
    let seen = Arc::new(Mutex::new(Vec::<String>::new()));
    let record = Arc::clone(&seen);
    let server = thread::spawn(move || {
        for answer in [
            "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 26\r\nConnection: close\r\n\r\n{\"success\":true,\"count\":2}",
            "HTTP/1.1 302 Found\r\nLocation: https://elsewhere.test/\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
        ] {
            let (mut stream, _) = listener.accept().unwrap();
            let mut reader = BufReader::new(stream.try_clone().unwrap());
            let mut head = String::new();
            let mut length = 0usize;
            loop {
                let mut line = String::new();
                reader.read_line(&mut line).unwrap();
                if line == "\r\n" || line.is_empty() {
                    break;
                }
                if let Some(value) = line.to_ascii_lowercase().strip_prefix("content-length:") {
                    length = value.trim().parse().unwrap();
                }
                head.push_str(&line);
            }
            let mut body = vec![0u8; length];
            reader.read_exact(&mut body).unwrap();
            record.lock().unwrap().push(format!("{head}{}", String::from_utf8_lossy(&body)));
            stream.write_all(answer.as_bytes()).unwrap();
        }
    });
    let client = Client::builder("mg_test_key")
        .base_url(format!("http://127.0.0.1:{port}"))
        .build();
    let got = client
        .contacts()
        .bulk_delete(
            "cb_1",
            &meeng::ContactsBulkDeleteParams::new(vec!["c1".to_owned()]),
        )
        .unwrap();
    assert_eq!(got.count, 2);
    let error = client.emails().cancel("em_1").unwrap_err();
    assert_eq!(error.api().map(|e| e.status), Some(302));
    server.join().unwrap();
    let seen = seen.lock().unwrap();
    assert!(
        seen[0].starts_with("DELETE /api/v1/contactBooks/cb_1/contacts/bulk HTTP/1.1"),
        "{}",
        seen[0]
    );
    assert!(
        seen[0].ends_with("{\"contactIds\":[\"c1\"]}"),
        "{}",
        seen[0]
    );
}

const SECRET: &str = "whsec_test";
const BODY: &str = r#"{"id":"evt_1","type":"email.delivered","createdAt":"2026-11-20T12:00:00Z","data":{"id":"em_1","status":"DELIVERED","from":"h@acme.com","to":["a@x.com"],"occurredAt":"2026-11-20T12:00:00Z"}}"#;

fn sign(timestamp: &str, body: &str, secret: &str) -> String {
    let mut mac = Hmac::<Sha256>::new_from_slice(secret.as_bytes()).unwrap();
    mac.update(format!("{timestamp}.{body}").as_bytes());
    let hex: String = mac
        .finalize()
        .into_bytes()
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect();
    format!("v1={hex}")
}

fn now_ms() -> i64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_millis() as i64
}

#[test]
fn a_signed_webhook_is_verified_and_read_as_a_typed_event() {
    let timestamp = now_ms().to_string();
    let webhooks = Client::new("k").webhooks(SECRET);
    let headers = [
        ("X-Meeng-Signature", sign(&timestamp, BODY, SECRET)),
        ("X-Meeng-Timestamp", timestamp.clone()),
    ];
    assert!(webhooks.verify(BODY.as_bytes(), headers.clone()));
    match webhooks.construct_event(BODY.as_bytes(), headers).unwrap() {
        WebhookEvent::Email(event) => {
            assert_eq!(event.type_, "email.delivered");
            assert_eq!(event.data.to, vec!["a@x.com".to_owned()]);
            assert_eq!(event.data.status.as_str(), "DELIVERED");
        }
        other => panic!("not an email event: {other:?}"),
    }
}

#[test]
fn the_usesend_headers_are_accepted_as_well_in_any_case() {
    let timestamp = now_ms().to_string();
    let headers = [
        ("x-usesend-signature", sign(&timestamp, BODY, SECRET)),
        ("X-UseSend-Timestamp", timestamp),
    ];
    assert!(Webhooks::new(SECRET).verify(BODY.as_bytes(), headers));
}

#[test]
fn a_webhook_is_refused_with_the_code_that_says_why() {
    let now: i64 = 1_800_000_000_000;
    let timestamp = now.to_string();
    let old = (now - 6 * 60 * 1000).to_string();
    let webhooks = Webhooks::new(SECRET);
    let refused = |headers: Vec<(&str, String)>, body: &str, options: WebhookOptions| -> String {
        let options = WebhookOptions {
            now_ms: Some(now),
            ..options
        };
        match webhooks.construct_event_with(body.as_bytes(), headers, &options) {
            Ok(_) => "ACCEPTED".to_owned(),
            Err(error) => error.code.to_owned(),
        }
    };
    let good = || {
        vec![
            ("X-Meeng-Signature", sign(&timestamp, BODY, SECRET)),
            ("X-Meeng-Timestamp", timestamp.clone()),
        ]
    };
    let stale = || {
        vec![
            ("X-Meeng-Signature", sign(&old, BODY, SECRET)),
            ("X-Meeng-Timestamp", old.clone()),
        ]
    };
    let default = WebhookOptions::default;
    assert_eq!(refused(good(), BODY, default()), "ACCEPTED");
    assert_eq!(
        refused(
            vec![("X-Meeng-Timestamp", timestamp.clone())],
            BODY,
            default()
        ),
        "MISSING_SIGNATURE"
    );
    assert_eq!(
        refused(
            vec![("X-Meeng-Signature", sign(&timestamp, BODY, SECRET))],
            BODY,
            default()
        ),
        "MISSING_TIMESTAMP"
    );
    assert_eq!(
        refused(
            vec![
                (
                    "X-Meeng-Signature",
                    sign(&timestamp, BODY, SECRET)[3..].to_owned()
                ),
                ("X-Meeng-Timestamp", timestamp.clone())
            ],
            BODY,
            default()
        ),
        "INVALID_SIGNATURE_FORMAT"
    );
    assert_eq!(
        refused(
            vec![
                ("X-Meeng-Signature", sign(&timestamp, BODY, SECRET)),
                ("X-Meeng-Timestamp", "yesterday".to_owned())
            ],
            BODY,
            default()
        ),
        "INVALID_TIMESTAMP"
    );
    assert_eq!(refused(stale(), BODY, default()), "TIMESTAMP_OUT_OF_RANGE");
    assert_eq!(
        refused(
            stale(),
            BODY,
            WebhookOptions {
                tolerance_ms: Some(-1),
                ..default()
            }
        ),
        "ACCEPTED"
    );
    assert_eq!(
        refused(
            stale(),
            BODY,
            WebhookOptions {
                tolerance_ms: Some(10 * 60 * 1000),
                ..default()
            }
        ),
        "ACCEPTED"
    );
    assert_eq!(
        refused(
            good(),
            BODY,
            WebhookOptions {
                secret: Some("another".to_owned()),
                ..default()
            }
        ),
        "SIGNATURE_MISMATCH"
    );
    assert_eq!(
        refused(good(), &BODY.replace("em_1", "em_2"), default()),
        "SIGNATURE_MISMATCH"
    );
    assert_eq!(
        refused(
            vec![
                ("X-Meeng-Signature", sign(&timestamp, "not json", SECRET)),
                ("X-Meeng-Timestamp", timestamp.clone())
            ],
            "not json",
            default()
        ),
        "INVALID_JSON"
    );
}
