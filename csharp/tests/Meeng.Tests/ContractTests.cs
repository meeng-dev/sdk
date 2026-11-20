using System;
using System.Collections.Generic;
using System.Threading.Tasks;
using Xunit;

namespace Meeng.Tests;

/// <summary>
/// The contract test calls every method of the SDK against a fake of the API that is
/// built from the API's contract and refuses what the contract does not allow.
/// check.sh starts the fake and sets MEENG_FAKE_URL.
/// </summary>
public class ContractTests
{
    /// <summary>A fact that is skipped, and says why, when there is no fake API to call.</summary>
    public sealed class FakeFactAttribute : FactAttribute
    {
        public FakeFactAttribute()
        {
            if (string.IsNullOrEmpty(Environment.GetEnvironmentVariable("MEENG_FAKE_URL")))
            {
                Skip = "MEENG_FAKE_URL is not set: start the fake API (go run ./cmd/fakeapi) to run the contract test";
            }
        }
    }

    private static MeengClient Client()
    {
        var fake = Environment.GetEnvironmentVariable("MEENG_FAKE_URL");
        return new MeengClient("mg_contract_test", new MeengClientOptions { BaseUrl = fake, RetryDelay = TimeSpan.FromMilliseconds(1) });
    }

    [FakeFact]
    public async Task EveryMethodMakesTheCallTheContractDescribes()
    {
        var meeng = Client();
        var failures = new List<string>();
        try { await meeng.Emails.SendAsync(new EmailsSendParams { To = ["hello@example.com"], From = "hello@example.com" }, new RequestOptions { IdempotencyKey = "contract-emails-send" }); } catch (Exception e) { failures.Add("emails.send: " + e.Message); }
        try { await meeng.Emails.BatchAsync([new EmailsBatchParams { To = ["hello@example.com"], From = "hello@example.com" }], new RequestOptions { IdempotencyKey = "contract-emails-batch" }); } catch (Exception e) { failures.Add("emails.batch: " + e.Message); }
        try { await meeng.Emails.GetAsync("id_emailid"); } catch (Exception e) { failures.Add("emails.get: " + e.Message); }
        try { await meeng.Emails.ListAsync(new EmailsListQuery { Page = 2, Limit = 10, StartDate = "2026-11-01T00:00:00Z", EndDate = "2026-11-30T00:00:00Z", DomainId = ["1", "2"] }); } catch (Exception e) { failures.Add("emails.list: " + e.Message); }
        try { await meeng.Emails.UpdateAsync("id_emailid", new EmailsUpdateParams { ScheduledAt = "2026-12-01T09:00:00Z" }); } catch (Exception e) { failures.Add("emails.update: " + e.Message); }
        try { await meeng.Emails.CancelAsync("id_emailid"); } catch (Exception e) { failures.Add("emails.cancel: " + e.Message); }
        try { await meeng.Domains.ListAsync(); } catch (Exception e) { failures.Add("domains.list: " + e.Message); }
        try { await meeng.Domains.CreateAsync(new DomainsCreateParams { Name = "example.com", Region = "us-east-1" }); } catch (Exception e) { failures.Add("domains.create: " + e.Message); }
        try { await meeng.Domains.VerifyAsync("id_id"); } catch (Exception e) { failures.Add("domains.verify: " + e.Message); }
        try { await meeng.Domains.GetAsync("id_id"); } catch (Exception e) { failures.Add("domains.get: " + e.Message); }
        try { await meeng.Domains.DeleteAsync("id_id"); } catch (Exception e) { failures.Add("domains.delete: " + e.Message); }
        try { await meeng.Contacts.CreateAsync("id_contactbookid", new ContactsCreateParams { Email = "hello@example.com" }); } catch (Exception e) { failures.Add("contacts.create: " + e.Message); }
        try { await meeng.Contacts.ListAsync("id_contactbookid", new ContactsListQuery { Emails = "a@example.com", Page = 2, Limit = 10, Ids = "c1,c2" }); } catch (Exception e) { failures.Add("contacts.list: " + e.Message); }
        try { await meeng.Contacts.GetAsync("id_contactbookid", "id_contactid"); } catch (Exception e) { failures.Add("contacts.get: " + e.Message); }
        try { await meeng.Contacts.UpdateAsync("id_contactbookid", "id_contactid", new ContactsUpdateParams()); } catch (Exception e) { failures.Add("contacts.update: " + e.Message); }
        try { await meeng.Contacts.UpsertAsync("id_contactbookid", "id_contactid", new ContactsUpsertParams { Email = "hello@example.com" }); } catch (Exception e) { failures.Add("contacts.upsert: " + e.Message); }
        try { await meeng.Contacts.DeleteAsync("id_contactbookid", "id_contactid"); } catch (Exception e) { failures.Add("contacts.delete: " + e.Message); }
        try { await meeng.Contacts.BulkCreateAsync("id_contactbookid", [new ContactsBulkCreateParams { Email = "hello@example.com" }]); } catch (Exception e) { failures.Add("contacts.bulkCreate: " + e.Message); }
        try { await meeng.Contacts.BulkDeleteAsync("id_contactbookid", new ContactsBulkDeleteParams { ContactIds = ["c1", "c2"] }); } catch (Exception e) { failures.Add("contacts.bulkDelete: " + e.Message); }
        try { await meeng.ContactBooks.ListAsync(); } catch (Exception e) { failures.Add("contactBooks.list: " + e.Message); }
        try { await meeng.ContactBooks.CreateAsync(new ContactBooksCreateParams { Name = "example.com" }); } catch (Exception e) { failures.Add("contactBooks.create: " + e.Message); }
        try { await meeng.ContactBooks.GetAsync("id_contactbookid"); } catch (Exception e) { failures.Add("contactBooks.get: " + e.Message); }
        try { await meeng.ContactBooks.UpdateAsync("id_contactbookid", new ContactBooksUpdateParams()); } catch (Exception e) { failures.Add("contactBooks.update: " + e.Message); }
        try { await meeng.ContactBooks.DeleteAsync("id_contactbookid"); } catch (Exception e) { failures.Add("contactBooks.delete: " + e.Message); }
        try { await meeng.Campaigns.CreateAsync(new CampaignsCreateParams { Name = "example.com", From = "hello@example.com", Subject = "sample", ContactBookId = "id_contactbookid" }); } catch (Exception e) { failures.Add("campaigns.create: " + e.Message); }
        try { await meeng.Campaigns.ListAsync(new CampaignsListQuery { Page = 2, Status = "DRAFT", Search = "news" }); } catch (Exception e) { failures.Add("campaigns.list: " + e.Message); }
        try { await meeng.Campaigns.GetAsync("id_campaignid"); } catch (Exception e) { failures.Add("campaigns.get: " + e.Message); }
        try { await meeng.Campaigns.DeleteAsync("id_campaignid"); } catch (Exception e) { failures.Add("campaigns.delete: " + e.Message); }
        try { await meeng.Campaigns.ScheduleAsync("id_campaignid", new CampaignsScheduleParams()); } catch (Exception e) { failures.Add("campaigns.schedule: " + e.Message); }
        try { await meeng.Campaigns.PauseAsync("id_campaignid"); } catch (Exception e) { failures.Add("campaigns.pause: " + e.Message); }
        try { await meeng.Campaigns.ResumeAsync("id_campaignid"); } catch (Exception e) { failures.Add("campaigns.resume: " + e.Message); }
        try { await meeng.Analytics.EmailTimeSeriesAsync(new AnalyticsEmailTimeSeriesQuery { Days = "7", DomainId = "1" }); } catch (Exception e) { failures.Add("analytics.emailTimeSeries: " + e.Message); }
        try { await meeng.Analytics.ReputationMetricsAsync(new AnalyticsReputationMetricsQuery { DomainId = "1" }); } catch (Exception e) { failures.Add("analytics.reputationMetrics: " + e.Message); }
        Assert.Empty(failures);
    }

    [FakeFact]
    public async Task ARefusalIsATypedError()
    {
        var refusal = await Assert.ThrowsAsync<MeengException>(() => Client().Emails.GetAsync("fake-404"));
        Assert.Equal(404, refusal.Status);
        Assert.Equal("NOT_FOUND", refusal.Code);
    }

    [FakeFact]
    public async Task ATooManyRequestsAnswerIsRepeatedUntilItPasses()
    {
        var meeng = Client();
        await meeng.Domains.GetAsync("flaky-429-2");
        var refusal = await Assert.ThrowsAsync<MeengException>(() => meeng.Domains.GetAsync("fake-429"));
        Assert.Equal(429, refusal.Status);
        Assert.Equal("RATE_LIMITED", refusal.Code);
    }

    [FakeFact]
    public async Task AServerFailureIsRepeatedForARead()
    {
        await Client().Emails.GetAsync("flaky-503-2");
    }

    [FakeFact]
    public async Task TheSameIdempotencyKeyAnswersTheSameEmail()
    {
        var meeng = Client();
        var options = new RequestOptions { IdempotencyKey = $"order-{DateTime.UtcNow.Ticks}" };
        var email = new EmailsSendParams { From = "hello@example.com", To = ["a@example.com"] };
        var first = await meeng.Emails.SendAsync(email, options);
        var again = await meeng.Emails.SendAsync(email, options);
        Assert.Equal(first.EmailId, again.EmailId);
        var refusal = await Assert.ThrowsAsync<MeengException>(() => meeng.Emails.SendAsync(new EmailsSendParams { From = "hello@example.com", To = ["b@example.com"] }, options));
        Assert.Equal(409, refusal.Status);
        Assert.Equal("NOT_UNIQUE", refusal.Code);
    }
}
