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
    private String version;
    private Long teamId;
    private Integer attempt;

    WebhookEvent(String id, String type, String createdAt) {
        this.id = id;
        this.type = type;
        this.createdAt = createdAt;
    }

    /** Sets what the delivery says about itself; the Webhooks handler calls it before it answers. */
    WebhookEvent withEnvelope(String version, Long teamId, Integer attempt) {
        this.version = version;
        this.teamId = teamId;
        this.attempt = attempt;
        return this;
    }

    /** @return the version of the payload's shape, such as 2026-01-18; null when an older server sent none */
    public String getVersion() {
        return version;
    }

    /** @return the team the event belongs to; null when an older server sent none */
    public Long getTeamId() {
        return teamId;
    }

    /** @return which delivery attempt this is, from 1 to 6; null when an older server sent none */
    public Integer getAttempt() {
        return attempt;
    }

    /** @return the id of the delivery: the same on every attempt, so a receiver can deduplicate by it */
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
