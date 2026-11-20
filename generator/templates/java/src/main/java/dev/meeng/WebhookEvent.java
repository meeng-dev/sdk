package dev.meeng;

/**
 * A webhook meeng sent, verified: one subclass a kind of data
 * ({@link EmailWebhookEvent} carries an {@link EmailBasePayload}…), or
 * {@link UnknownWebhookEvent} for a type this SDK does not know yet. Within a
 * class, {@link #getType()} says what happened.
 */
public abstract class WebhookEvent {
    private final String id;
    private final String type;
    private final String createdAt;

    WebhookEvent(String id, String type, String createdAt) {
        this.id = id;
        this.type = type;
        this.createdAt = createdAt;
    }

    /** @return the event's id */
    public String getId() {
        return id;
    }

    /** @return what happened: email.delivered… */
    public String getType() {
        return type;
    }

    /** @return when the event was created, RFC 3339 */
    public String getCreatedAt() {
        return createdAt;
    }

    /** @return what the event carries */
    public abstract Object getData();

    @Override
    public String toString() {
        return getClass().getSimpleName() + "{id=" + id + ", type=" + type + ", createdAt=" + createdAt + ", data=" + getData() + "}";
    }
}
