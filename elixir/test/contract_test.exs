# The contract test calls every method of the SDK against a fake of the API that is
# built from the API's contract and refuses what the contract does not allow.
# check.sh starts the fake and sets MEENG_FAKE_URL.

defmodule ContractTest do
  use ExUnit.Case, async: false

  @fake System.get_env("MEENG_FAKE_URL")

  if @fake in [nil, ""] do
    @moduletag skip: "MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test"
  end

  setup do
    {:ok, meeng: Meeng.new("mg_contract_test", base_url: @fake, retry_delay: 1)}
  end

  test "every method makes the call the contract describes", %{meeng: meeng} do
    assert {:ok, _} = Meeng.Emails.send(meeng, %{"to" => ["hello@example.com"], "from" => "hello@example.com"}, idempotency_key: "contract-emails-send"), "emails.send"
    assert {:ok, _} = Meeng.Emails.batch(meeng, [%{"to" => ["hello@example.com"], "from" => "hello@example.com"}], idempotency_key: "contract-emails-batch"), "emails.batch"
    assert {:ok, _} = Meeng.Emails.get(meeng, "id_emailid"), "emails.get"
    assert {:ok, _} = Meeng.Emails.list(meeng, page: 2, limit: 10, start_date: "2026-11-01T00:00:00Z", end_date: "2026-11-30T00:00:00Z", domain_id: ["1", "2"]), "emails.list"
    assert {:ok, _} = Meeng.Emails.update(meeng, "id_emailid", %{"scheduledAt" => "2026-12-01T09:00:00Z"}), "emails.update"
    assert {:ok, _} = Meeng.Emails.cancel(meeng, "id_emailid"), "emails.cancel"
    assert {:ok, _} = Meeng.Domains.list(meeng), "domains.list"
    assert {:ok, _} = Meeng.Domains.create(meeng, %{"name" => "example.com", "region" => "us-east-1"}), "domains.create"
    assert {:ok, _} = Meeng.Domains.verify(meeng, "id_id"), "domains.verify"
    assert {:ok, _} = Meeng.Domains.get(meeng, "id_id"), "domains.get"
    assert {:ok, _} = Meeng.Domains.delete(meeng, "id_id"), "domains.delete"
    assert {:ok, _} = Meeng.Contacts.create(meeng, "id_contactbookid", %{"email" => "hello@example.com"}), "contacts.create"
    assert {:ok, _} = Meeng.Contacts.list(meeng, "id_contactbookid", emails: "a@example.com", page: 2, limit: 10, ids: "c1,c2"), "contacts.list"
    assert {:ok, _} = Meeng.Contacts.get(meeng, "id_contactbookid", "id_contactid"), "contacts.get"
    assert {:ok, _} = Meeng.Contacts.update(meeng, "id_contactbookid", "id_contactid", %{}), "contacts.update"
    assert {:ok, _} = Meeng.Contacts.upsert(meeng, "id_contactbookid", "id_contactid", %{"email" => "hello@example.com"}), "contacts.upsert"
    assert {:ok, _} = Meeng.Contacts.delete(meeng, "id_contactbookid", "id_contactid"), "contacts.delete"
    assert {:ok, _} = Meeng.Contacts.bulk_create(meeng, "id_contactbookid", [%{"email" => "hello@example.com"}]), "contacts.bulkCreate"
    assert {:ok, _} = Meeng.Contacts.bulk_delete(meeng, "id_contactbookid", %{"contactIds" => ["c1", "c2"]}), "contacts.bulkDelete"
    assert {:ok, _} = Meeng.ContactBooks.list(meeng), "contactBooks.list"
    assert {:ok, _} = Meeng.ContactBooks.create(meeng, %{"name" => "example.com"}), "contactBooks.create"
    assert {:ok, _} = Meeng.ContactBooks.get(meeng, "id_contactbookid"), "contactBooks.get"
    assert {:ok, _} = Meeng.ContactBooks.update(meeng, "id_contactbookid", %{}), "contactBooks.update"
    assert {:ok, _} = Meeng.ContactBooks.delete(meeng, "id_contactbookid"), "contactBooks.delete"
    assert {:ok, _} = Meeng.Campaigns.create(meeng, %{"name" => "example.com", "from" => "hello@example.com", "subject" => "sample", "contactBookId" => "id_contactbookid"}), "campaigns.create"
    assert {:ok, _} = Meeng.Campaigns.list(meeng, page: 2, status: "DRAFT", search: "news"), "campaigns.list"
    assert {:ok, _} = Meeng.Campaigns.get(meeng, "id_campaignid"), "campaigns.get"
    assert {:ok, _} = Meeng.Campaigns.delete(meeng, "id_campaignid"), "campaigns.delete"
    assert {:ok, _} = Meeng.Campaigns.schedule(meeng, "id_campaignid", %{}), "campaigns.schedule"
    assert {:ok, _} = Meeng.Campaigns.pause(meeng, "id_campaignid"), "campaigns.pause"
    assert {:ok, _} = Meeng.Campaigns.resume(meeng, "id_campaignid"), "campaigns.resume"
    assert {:ok, _} = Meeng.Analytics.email_time_series(meeng, days: "7", domain_id: "1"), "analytics.emailTimeSeries"
    assert {:ok, _} = Meeng.Analytics.reputation_metrics(meeng, domain_id: "1"), "analytics.reputationMetrics"
  end

  test "a refusal is a typed error", %{meeng: meeng} do
    assert {:error, %Meeng.Error{status: 404, code: "NOT_FOUND"}} = Meeng.Emails.get(meeng, "fake-404")
  end

  test "a too many requests answer is repeated until it passes", %{meeng: meeng} do
    assert {:ok, _} = Meeng.Domains.get(meeng, "flaky-429-2")
    assert {:error, %Meeng.Error{status: 429, code: "RATE_LIMITED"}} = Meeng.Domains.get(meeng, "fake-429")
  end

  test "a server failure is repeated for a read", %{meeng: meeng} do
    assert {:ok, _} = Meeng.Emails.get(meeng, "flaky-503-2")
  end

  test "the same idempotency key answers the same email", %{meeng: meeng} do
    key = "order-#{System.unique_integer([:positive])}-#{System.os_time(:nanosecond)}"
    email = %{"from" => "hello@example.com", "to" => ["a@example.com"]}
    assert {:ok, %{"emailId" => id}} = Meeng.Emails.send(meeng, email, idempotency_key: key)
    assert {:ok, %{"emailId" => ^id}} = Meeng.Emails.send(meeng, email, idempotency_key: key)

    other = %{email | "to" => ["b@example.com"]}
    assert {:error, %Meeng.Error{status: 409, code: "NOT_UNIQUE"}} = Meeng.Emails.send(meeng, other, idempotency_key: key)
  end

  test "an email needs its required fields", %{meeng: meeng} do
    assert {:error, %Meeng.Error{status: 400, code: "ZodError"}} = Meeng.Emails.send(meeng, %{"to" => "a@example.com"})
  end
end
