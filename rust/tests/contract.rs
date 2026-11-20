//! The contract test calls every method of the SDK against a fake of the API that
//! is built from the API's contract and refuses what the contract does not allow.
//! check.sh starts the fake and sets MEENG_FAKE_URL.

// Written by the generator, as the modules of the SDK are: not for rustfmt.
#![cfg_attr(rustfmt, rustfmt::skip)]

use std::time::{Duration, SystemTime, UNIX_EPOCH};

use meeng::{
    Client, Error,
    AnalyticsEmailTimeSeriesQuery,
    AnalyticsReputationMetricsQuery,
    CampaignsCreateParams,
    CampaignsListQuery,
    CampaignsScheduleParams,
    ContactBooksCreateParams,
    ContactBooksUpdateParams,
    ContactsBulkCreateParams,
    ContactsBulkDeleteParams,
    ContactsCreateParams,
    ContactsListQuery,
    ContactsUpdateParams,
    ContactsUpsertParams,
    DomainsCreateParams,
    EmailsBatchParams,
    EmailsListQuery,
    EmailsSendParams,
    EmailsUpdateParams,
};

fn fake() -> Option<Client> {
    let base = std::env::var("MEENG_FAKE_URL").ok().filter(|value| !value.is_empty());
    if base.is_none() {
        eprintln!("MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test");
    }
    base.map(|base| {
        Client::builder("mg_contract_test")
            .base_url(base)
            .retry_delay(Duration::from_millis(1))
            .build()
    })
}

fn refusal(error: Error) -> (u16, String) {
    match error {
        Error::Api(api) => (api.status, api.code),
        other => panic!("not a refusal of the API: {other}"),
    }
}

#[test]
fn every_method_makes_the_call_the_contract_describes() {
    let Some(client) = fake() else { return };
    let mut failures = Vec::new();
    if let Err(error) = client.emails().idempotency_key("contract-emails-send").send(&EmailsSendParams::new(vec!["hello@example.com".to_owned()], "hello@example.com")) {
        failures.push(format!("emails.send: {error}"));
    }
    if let Err(error) = client.emails().idempotency_key("contract-emails-batch").batch(&[EmailsBatchParams::new(vec!["hello@example.com".to_owned()], "hello@example.com")]) {
        failures.push(format!("emails.batch: {error}"));
    }
    if let Err(error) = client.emails().get("id_emailid") {
        failures.push(format!("emails.get: {error}"));
    }
    if let Err(error) = client.emails().list(&EmailsListQuery::new().page(2).limit(10).start_date("2026-11-01T00:00:00Z").end_date("2026-11-30T00:00:00Z").domain_id(["1", "2"])) {
        failures.push(format!("emails.list: {error}"));
    }
    if let Err(error) = client.emails().update("id_emailid", &EmailsUpdateParams::new("2026-12-01T09:00:00Z")) {
        failures.push(format!("emails.update: {error}"));
    }
    if let Err(error) = client.emails().cancel("id_emailid") {
        failures.push(format!("emails.cancel: {error}"));
    }
    if let Err(error) = client.domains().list() {
        failures.push(format!("domains.list: {error}"));
    }
    if let Err(error) = client.domains().create(&DomainsCreateParams::new("example.com", "us-east-1")) {
        failures.push(format!("domains.create: {error}"));
    }
    if let Err(error) = client.domains().verify("id_id") {
        failures.push(format!("domains.verify: {error}"));
    }
    if let Err(error) = client.domains().get("id_id") {
        failures.push(format!("domains.get: {error}"));
    }
    if let Err(error) = client.domains().delete("id_id") {
        failures.push(format!("domains.delete: {error}"));
    }
    if let Err(error) = client.contacts().create("id_contactbookid", &ContactsCreateParams::new("hello@example.com")) {
        failures.push(format!("contacts.create: {error}"));
    }
    if let Err(error) = client.contacts().list("id_contactbookid", &ContactsListQuery::new().emails("a@example.com").page(2).limit(10).ids("c1,c2")) {
        failures.push(format!("contacts.list: {error}"));
    }
    if let Err(error) = client.contacts().get("id_contactbookid", "id_contactid") {
        failures.push(format!("contacts.get: {error}"));
    }
    if let Err(error) = client.contacts().update("id_contactbookid", "id_contactid", &ContactsUpdateParams::new()) {
        failures.push(format!("contacts.update: {error}"));
    }
    if let Err(error) = client.contacts().upsert("id_contactbookid", "id_contactid", &ContactsUpsertParams::new("hello@example.com")) {
        failures.push(format!("contacts.upsert: {error}"));
    }
    if let Err(error) = client.contacts().delete("id_contactbookid", "id_contactid") {
        failures.push(format!("contacts.delete: {error}"));
    }
    if let Err(error) = client.contacts().bulk_create("id_contactbookid", &[ContactsBulkCreateParams::new("hello@example.com")]) {
        failures.push(format!("contacts.bulkCreate: {error}"));
    }
    if let Err(error) = client.contacts().bulk_delete("id_contactbookid", &ContactsBulkDeleteParams::new(vec!["c1".to_owned(), "c2".to_owned()])) {
        failures.push(format!("contacts.bulkDelete: {error}"));
    }
    if let Err(error) = client.contact_books().list() {
        failures.push(format!("contactBooks.list: {error}"));
    }
    if let Err(error) = client.contact_books().create(&ContactBooksCreateParams::new("example.com")) {
        failures.push(format!("contactBooks.create: {error}"));
    }
    if let Err(error) = client.contact_books().get("id_contactbookid") {
        failures.push(format!("contactBooks.get: {error}"));
    }
    if let Err(error) = client.contact_books().update("id_contactbookid", &ContactBooksUpdateParams::new()) {
        failures.push(format!("contactBooks.update: {error}"));
    }
    if let Err(error) = client.contact_books().delete("id_contactbookid") {
        failures.push(format!("contactBooks.delete: {error}"));
    }
    if let Err(error) = client.campaigns().create(&CampaignsCreateParams::new("example.com", "hello@example.com", "sample", "id_contactbookid")) {
        failures.push(format!("campaigns.create: {error}"));
    }
    if let Err(error) = client.campaigns().list(&CampaignsListQuery::new().page(2).status("DRAFT").search("news")) {
        failures.push(format!("campaigns.list: {error}"));
    }
    if let Err(error) = client.campaigns().get("id_campaignid") {
        failures.push(format!("campaigns.get: {error}"));
    }
    if let Err(error) = client.campaigns().delete("id_campaignid") {
        failures.push(format!("campaigns.delete: {error}"));
    }
    if let Err(error) = client.campaigns().schedule("id_campaignid", &CampaignsScheduleParams::new()) {
        failures.push(format!("campaigns.schedule: {error}"));
    }
    if let Err(error) = client.campaigns().pause("id_campaignid") {
        failures.push(format!("campaigns.pause: {error}"));
    }
    if let Err(error) = client.campaigns().resume("id_campaignid") {
        failures.push(format!("campaigns.resume: {error}"));
    }
    if let Err(error) = client.analytics().email_time_series(&AnalyticsEmailTimeSeriesQuery::new().days("7").domain_id("1")) {
        failures.push(format!("analytics.emailTimeSeries: {error}"));
    }
    if let Err(error) = client.analytics().reputation_metrics(&AnalyticsReputationMetricsQuery::new().domain_id("1")) {
        failures.push(format!("analytics.reputationMetrics: {error}"));
    }
    assert!(failures.is_empty(), "{failures:#?}");
}

#[test]
fn a_refusal_is_a_typed_error() {
    let Some(client) = fake() else { return };
    let error = client.emails().get("fake-404").unwrap_err();
    assert_eq!(refusal(error), (404, "NOT_FOUND".to_owned()));
}

#[test]
fn a_too_many_requests_answer_is_repeated_until_it_passes() {
    let Some(client) = fake() else { return };
    client.domains().get("flaky-429-2").unwrap();
    let error = client.domains().get("fake-429").unwrap_err();
    assert_eq!(refusal(error), (429, "RATE_LIMITED".to_owned()));
}

#[test]
fn a_server_failure_is_repeated_for_a_read() {
    let Some(client) = fake() else { return };
    client.emails().get("flaky-503-2").unwrap();
}

#[test]
fn the_same_idempotency_key_answers_the_same_email() {
    let Some(client) = fake() else { return };
    let nanos = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_nanos();
    let key = format!("order-{nanos}");
    let email = EmailsSendParams::new(vec!["a@example.com".to_owned()], "hello@example.com");
    let first = client.emails().idempotency_key(key.as_str()).send(&email).unwrap();
    let again = client.emails().idempotency_key(key.as_str()).send(&email).unwrap();
    assert_eq!(first.email_id, again.email_id);
    let other = EmailsSendParams::new(vec!["b@example.com".to_owned()], "hello@example.com");
    let error = client.emails().idempotency_key(key.as_str()).send(&other).unwrap_err();
    assert_eq!(refusal(error), (409, "NOT_UNIQUE".to_owned()));
}
