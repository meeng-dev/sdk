package generator

// The contract does not describe the webhooks the API posts, so their events
// are written here, from the payloads the API sends (the useSend webhook
// events, which meeng reproduces): every event is {id, type, createdAt, data},
// and what data holds depends on the type.

func str(wire string, required bool) *Field {
	return &Field{Wire: wire, Required: required, Type: &TypeRef{Kind: "string"}}
}

func nullableStr(wire string) *Field {
	f := str(wire, false)
	f.Nullable = true
	return f
}

func enumStr(wire string, required bool, values ...string) *Field {
	return &Field{Wire: wire, Required: required, Type: &TypeRef{Kind: "string", Enum: values}}
}

func modelField(wire, model string, required bool) *Field {
	return &Field{Wire: wire, Required: required, Type: &TypeRef{Kind: "model", Model: model}}
}

func anyMap(wire string, required bool) *Field {
	return &Field{Wire: wire, Required: required, Type: &TypeRef{Kind: "map", Items: &TypeRef{Kind: "any"}}}
}

func flag(wire string, required bool) *Field {
	return &Field{Wire: wire, Required: required, Type: &TypeRef{Kind: "boolean"}}
}

func integer(wire string, required, nullable bool) *Field {
	return &Field{Wire: wire, Required: required, Nullable: nullable, Type: &TypeRef{Kind: "integer"}}
}

var emailStatuses = []string{"QUEUED", "SENT", "DELIVERY_DELAYED", "DELIVERED", "BOUNCED", "REJECTED", "RENDERING_FAILURE", "COMPLAINED", "FAILED", "CANCELLED", "SUPPRESSED", "OPENED", "CLICKED", "SCHEDULED"}

// emailBase are the fields every email event carries.
func emailBase() []*Field {
	to := &Field{Wire: "to", Required: true, Type: &TypeRef{Kind: "array", Items: &TypeRef{Kind: "string"}}}
	campaign := nullableStr("campaignId")
	contact := nullableStr("contactId")
	return []*Field{
		str("id", true),
		enumStr("status", true, emailStatuses...),
		str("from", true),
		to,
		str("occurredAt", true),
		campaign,
		contact,
		integer("domainId", false, true),
		str("subject", false),
		str("templateId", false),
		anyMap("metadata", false),
	}
}

func emailWith(name, description string, extra ...*Field) *Model {
	return &Model{Name: name, Description: description, Fields: append(emailBase(), extra...)}
}

// webhookSurface is the events the API posts, grouped by payload, and the
// models those payloads are.
func webhookSurface() ([]*WebhookGroup, []*Model, error) {
	models := []*Model{
		emailWith("EmailBasePayload", "What most email events carry: the email, its status and when it happened."),
		emailWith("EmailBouncedPayload", "The data of email.bounced: the email and how it bounced.", modelField("bounce", "EmailBouncedBounce", true)),
		{Name: "EmailBouncedBounce", Description: "How an email bounced.", Fields: []*Field{
			enumStr("type", true, "Transient", "Permanent", "Undetermined"),
			enumStr("subType", true, "General", "NoEmail", "Suppressed", "OnAccountSuppressionList", "MailboxFull", "MessageTooLarge", "ContentRejected", "AttachmentRejected"),
			str("message", false),
		}},
		emailWith("EmailFailedPayload", "The data of email.failed: the email and why it failed.", modelField("failed", "EmailFailedFailure", true)),
		{Name: "EmailFailedFailure", Description: "Why an email failed.", Fields: []*Field{str("reason", true)}},
		emailWith("EmailSuppressedPayload", "The data of email.suppressed: the email and why it was not sent.", modelField("suppression", "EmailSuppressedSuppression", true)),
		{Name: "EmailSuppressedSuppression", Description: "Why an email was suppressed.", Fields: []*Field{
			enumStr("type", true, "Bounce", "Complaint", "Manual"),
			str("reason", true),
			str("source", false),
		}},
		emailWith("EmailOpenedPayload", "The data of email.opened: the email and the open.", modelField("open", "EmailOpenedOpen", true)),
		{Name: "EmailOpenedOpen", Description: "One open of an email.", Fields: []*Field{
			str("timestamp", true), str("userAgent", false), str("ip", false), str("platform", false),
		}},
		emailWith("EmailClickedPayload", "The data of email.clicked: the email and the click.", modelField("click", "EmailClickedClick", true)),
		{Name: "EmailClickedClick", Description: "One click on a link of an email.", Fields: []*Field{
			str("timestamp", true), str("url", true), str("userAgent", false), str("ip", false), str("platform", false),
		}},
		{Name: "ContactPayload", Description: "The data of the contact events.", Fields: []*Field{
			str("id", true), str("email", true), str("contactBookId", true), flag("subscribed", true), anyMap("properties", true),
			nullableStr("firstName"), nullableStr("lastName"), str("createdAt", true), str("updatedAt", true),
		}},
		{Name: "DomainPayload", Description: "The data of the domain events.", Fields: []*Field{
			integer("id", true, false), str("name", true), str("status", true), str("region", true), str("createdAt", true), str("updatedAt", true),
			flag("clickTracking", true), flag("openTracking", true), nullableStr("subdomain"), nullableStr("sesTenantId"), nullableStr("dkimStatus"), nullableStr("spfDetails"),
			{Wire: "dmarcAdded", Nullable: true, Type: &TypeRef{Kind: "boolean"}},
		}},
		{Name: "WebhookTestPayload", Description: "The data of webhook.test, which the dashboard sends to try an endpoint.", Fields: []*Field{
			flag("test", true), str("webhookId", true), str("sentAt", true),
		}},
	}
	for _, model := range models {
		for _, field := range model.Fields {
			if err := checkIdentifier("field of "+model.Name, field.Wire); err != nil {
				return nil, nil, err
			}
		}
	}
	group := func(name, data string, types ...string) *WebhookGroup {
		return &WebhookGroup{Name: name, Data: &TypeRef{Kind: "model", Model: data}, Types: types}
	}
	groups := []*WebhookGroup{
		group("Contact", "ContactPayload", "contact.created", "contact.updated", "contact.deleted"),
		group("Domain", "DomainPayload", "domain.created", "domain.verified", "domain.updated", "domain.deleted"),
		group("Email", "EmailBasePayload", "email.queued", "email.sent", "email.delivery_delayed", "email.delivered", "email.rejected", "email.rendering_failure", "email.complained", "email.cancelled"),
		group("EmailBounced", "EmailBouncedPayload", "email.bounced"),
		group("EmailClicked", "EmailClickedPayload", "email.clicked"),
		group("EmailFailed", "EmailFailedPayload", "email.failed"),
		group("EmailOpened", "EmailOpenedPayload", "email.opened"),
		group("EmailSuppressed", "EmailSuppressedPayload", "email.suppressed"),
		group("Test", "WebhookTestPayload", "webhook.test"),
	}
	for _, g := range groups {
		for _, t := range g.Types {
			if err := checkMatch("webhook", t, eventType); err != nil {
				return nil, nil, err
			}
		}
	}
	return groups, models, nil
}
