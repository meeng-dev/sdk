package dev.meeng;

/**
 * A request meeng did not sign, signed too long ago, or that is not an event.
 * {@link #getCode()} is MISSING_SIGNATURE, MISSING_TIMESTAMP,
 * INVALID_SIGNATURE_FORMAT, INVALID_TIMESTAMP, TIMESTAMP_OUT_OF_RANGE,
 * SIGNATURE_MISMATCH, INVALID_BODY or INVALID_JSON.
 */
public final class WebhookVerificationException extends RuntimeException {
    private static final long serialVersionUID = 1L;

    private final String code;

    /**
     * The refusal of a webhook.
     *
     * @param code why it was refused
     * @param message what is wrong
     */
    public WebhookVerificationException(String code, String message) {
        super(code + ": " + message);
        this.code = code;
    }

    /** @return why the webhook was refused: SIGNATURE_MISMATCH… */
    public String getCode() {
        return code;
    }
}
