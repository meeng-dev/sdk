package dev.meeng;

/** Options for one send: the idempotency key. */
public final class RequestOptions {
    private final String idempotencyKey;

    private RequestOptions(Builder builder) {
        this.idempotencyKey = builder.idempotencyKey;
    }

    /** @return a builder of options */
    public static Builder builder() {
        return new Builder();
    }

    /** @return the idempotency key, or null */
    public String getIdempotencyKey() {
        return idempotencyKey;
    }

    /** Builds {@link RequestOptions}. */
    public static final class Builder {
        private String idempotencyKey;

        private Builder() {
        }

        /**
         * Makes a send safe to retry: the same key with the same body answers
         * the first result again; with another body, a 409 NOT_UNIQUE. The key
         * is up to 256 characters and expires after 24 hours.
         *
         * @param idempotencyKey the key
         * @return this builder
         */
        public Builder idempotencyKey(String idempotencyKey) {
            this.idempotencyKey = idempotencyKey;
            return this;
        }

        /** @return the options */
        public RequestOptions build() {
            return new RequestOptions(this);
        }
    }
}
