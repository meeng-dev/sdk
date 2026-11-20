package dev.meeng;

/** A request meeng did not sign, or signed too long ago. */
public final class WebhookVerificationException extends RuntimeException {
    private static final long serialVersionUID = 1L;

    /** The refusal of a webhook. */
    public WebhookVerificationException() {
        super("meeng: webhook signature does not verify");
    }
}
