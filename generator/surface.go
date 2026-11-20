package generator

// The contract has no operationId, no tag and no named schema, so the names
// every SDK gives to its services, methods and models are written here, once.
// An operation the contract has that is not here stops generation, and so does
// one here that the contract lacks: the SDKs never silently drop or rename a
// route.

// operationDef names one operation.
type operationDef struct {
	// Key is "METHOD /path", as the contract writes them.
	Key string
	// Service is the resource: emails, domains, contacts, contactBooks,
	// campaigns, analytics. The clients expose it under this name.
	Service string
	// Name is the method: send, list, bulkCreate.
	Name string
	// Summary is one sentence for the doc comment, without a full stop.
	Summary string
	// Response names the model the answer is read into, or each item of the
	// list it is; empty means <Service><Name>Response. Operations that answer
	// the same shape share a name, and generation fails if two shapes differ.
	Response string
	// Aliases are other names the dynamic clients give the same method, as
	// useSend's SDKs do: emails.create for emails.send.
	Aliases []string
}

func (d operationDef) responseName() string {
	if d.Response != "" {
		return d.Response
	}
	return Pascal(d.Service) + Pascal(d.Name) + "Response"
}

// operations is the API's surface, in the order the SDKs list it.
var operations = []operationDef{
	{Key: "POST /v1/emails", Service: "emails", Name: "send", Summary: "Sends an email, now or at scheduledAt", Aliases: []string{"create"}},
	{Key: "POST /v1/emails/batch", Service: "emails", Name: "batch", Summary: "Sends up to 100 emails in one request"},
	{Key: "GET /v1/emails/{emailId}", Service: "emails", Name: "get", Summary: "Reads one email with its events", Response: "Email"},
	{Key: "GET /v1/emails", Service: "emails", Name: "list", Summary: "Lists emails, newest first, with an optional date range and domain filter"},
	{Key: "PATCH /v1/emails/{emailId}", Service: "emails", Name: "update", Summary: "Reschedules an email that has not been sent"},
	{Key: "POST /v1/emails/{emailId}/cancel", Service: "emails", Name: "cancel", Summary: "Cancels a scheduled email"},

	{Key: "GET /v1/domains", Service: "domains", Name: "list", Summary: "Lists the sending domains of the team", Response: "Domain"},
	{Key: "POST /v1/domains", Service: "domains", Name: "create", Summary: "Adds a sending domain and returns the DNS records to publish", Response: "Domain"},
	{Key: "PUT /v1/domains/{id}/verify", Service: "domains", Name: "verify", Summary: "Asks for the domain's DNS records to be checked again"},
	{Key: "GET /v1/domains/{id}", Service: "domains", Name: "get", Summary: "Reads one domain with its DNS records", Response: "Domain"},
	{Key: "DELETE /v1/domains/{id}", Service: "domains", Name: "delete", Summary: "Removes a domain"},

	{Key: "POST /v1/contactBooks/{contactBookId}/contacts", Service: "contacts", Name: "create", Summary: "Adds a contact to a contact book"},
	{Key: "GET /v1/contactBooks/{contactBookId}/contacts", Service: "contacts", Name: "list", Summary: "Lists the contacts of a contact book, optionally by email or id", Response: "Contact"},
	{Key: "GET /v1/contactBooks/{contactBookId}/contacts/{contactId}", Service: "contacts", Name: "get", Summary: "Reads one contact", Response: "Contact"},
	{Key: "PATCH /v1/contactBooks/{contactBookId}/contacts/{contactId}", Service: "contacts", Name: "update", Summary: "Changes the fields of a contact that are sent"},
	{Key: "PUT /v1/contactBooks/{contactBookId}/contacts/{contactId}", Service: "contacts", Name: "upsert", Summary: "Creates the contact, or replaces it when it exists"},
	{Key: "DELETE /v1/contactBooks/{contactBookId}/contacts/{contactId}", Service: "contacts", Name: "delete", Summary: "Removes a contact"},
	{Key: "POST /v1/contactBooks/{contactBookId}/contacts/bulk", Service: "contacts", Name: "bulkCreate", Summary: "Adds many contacts to a contact book in one request"},
	{Key: "DELETE /v1/contactBooks/{contactBookId}/contacts/bulk", Service: "contacts", Name: "bulkDelete", Summary: "Removes many contacts from a contact book in one request"},

	{Key: "GET /v1/contactBooks", Service: "contactBooks", Name: "list", Summary: "Lists the contact books of the team", Response: "ContactBook"},
	{Key: "POST /v1/contactBooks", Service: "contactBooks", Name: "create", Summary: "Creates a contact book", Response: "ContactBook"},
	{Key: "GET /v1/contactBooks/{contactBookId}", Service: "contactBooks", Name: "get", Summary: "Reads one contact book", Response: "ContactBook"},
	{Key: "PATCH /v1/contactBooks/{contactBookId}", Service: "contactBooks", Name: "update", Summary: "Changes the fields of a contact book that are sent", Response: "ContactBook"},
	{Key: "DELETE /v1/contactBooks/{contactBookId}", Service: "contactBooks", Name: "delete", Summary: "Removes a contact book and its contacts"},

	{Key: "POST /v1/campaigns", Service: "campaigns", Name: "create", Summary: "Creates a campaign, and sends it now when sendNow is true", Response: "Campaign"},
	{Key: "GET /v1/campaigns", Service: "campaigns", Name: "list", Summary: "Lists campaigns, optionally by status or a search text", Aliases: []string{"getAll"}},
	{Key: "GET /v1/campaigns/{campaignId}", Service: "campaigns", Name: "get", Summary: "Reads one campaign with its counters", Response: "Campaign"},
	{Key: "DELETE /v1/campaigns/{campaignId}", Service: "campaigns", Name: "delete", Summary: "Removes a campaign", Response: "Campaign"},
	{Key: "POST /v1/campaigns/{campaignId}/schedule", Service: "campaigns", Name: "schedule", Summary: "Schedules a campaign, or sends it now when no time is given"},
	{Key: "POST /v1/campaigns/{campaignId}/pause", Service: "campaigns", Name: "pause", Summary: "Pauses a running or scheduled campaign", Response: "CampaignActionResponse"},
	{Key: "POST /v1/campaigns/{campaignId}/resume", Service: "campaigns", Name: "resume", Summary: "Resumes a paused campaign", Response: "CampaignActionResponse"},

	{Key: "GET /v1/analytics/email-time-series", Service: "analytics", Name: "emailTimeSeries", Summary: "Counts emails by day over the last 7 or 30 days"},
	{Key: "GET /v1/analytics/reputation-metrics", Service: "analytics", Name: "reputationMetrics", Summary: "Reads the bounce and complaint rates of the sending reputation"},
}

// surface is operations by key.
var surface = func() map[string]operationDef {
	out := make(map[string]operationDef, len(operations))
	for _, op := range operations {
		out[op.Key] = op
	}
	return out
}()

func serviceRank(name string) int {
	for i, op := range operations {
		if op.Service == name {
			return i
		}
	}
	return len(operations)
}

func methodRank(service, name string) int {
	for i, op := range operations {
		if op.Service == service && op.Name == name {
			return i
		}
	}
	return len(operations)
}

// queryDocs describe the filters the contract leaves without a description.
var queryDocs = map[string]string{
	"page":      "The page, from 1.",
	"limit":     "The items a page: 1 to 100, 50 by default.",
	"startDate": "Only emails created at or after this time (RFC 3339).",
	"endDate":   "Only emails created at or before this time (RFC 3339).",
	"domainId":  "Only emails of these domains; a list sends one domainId for each value.",
	"emails":    "Only the contacts with these emails, separated by commas.",
	"ids":       "Only the contacts with these ids, separated by commas.",
	"status":    "Only campaigns in this status: DRAFT, SCHEDULED, RUNNING, PAUSED or SENT.",
	"search":    "Only campaigns whose name matches this text.",
	"days":      "How many days to count: 7 or 30.",
}
