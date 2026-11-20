package dev.meeng;

import java.nio.charset.StandardCharsets;
import java.util.Map;

/**
 * A request the API refused. {@link #getCode()} is the API's, such as
 * NOT_UNIQUE, FORBIDDEN or NOT_FOUND; it is INTERNAL_SERVER_ERROR when the
 * answer carried none. {@link #getRetryAfter()} is how many seconds to wait on
 * a 429.
 */
public class MeengException extends RuntimeException {
    private static final long serialVersionUID = 1L;

    private final int status;
    private final String code;
    private final String apiMessage;
    private final Integer retryAfter;

    /**
     * A refusal.
     *
     * @param status the HTTP status
     * @param code the API's error code
     * @param apiMessage the API's message
     * @param retryAfter seconds to wait before trying again, or null
     */
    public MeengException(int status, String code, String apiMessage, Integer retryAfter) {
        super(apiMessage.isEmpty() ? "meeng: " + status + " " + code : "meeng: " + status + " " + code + ": " + apiMessage);
        this.status = status;
        this.code = code;
        this.apiMessage = apiMessage;
        this.retryAfter = retryAfter;
    }

    /** @return the HTTP status */
    public int getStatus() {
        return status;
    }

    /** @return the API's error code: NOT_UNIQUE, FORBIDDEN… */
    public String getCode() {
        return code;
    }

    /** @return the API's message */
    public String getApiMessage() {
        return apiMessage;
    }

    /** @return on a 429, how many seconds to wait; null otherwise */
    public Integer getRetryAfter() {
        return retryAfter;
    }

    /**
     * The refusal a response carries: the answer's {"error": {"code",
     * "message"}}, {"error": "text"} or the error itself. A body that is not
     * the API's JSON is still a refusal.
     */
    static MeengException of(Transport.Response response) {
        String code = "";
        String message = "";
        try {
            Object parsed = Json.parse(new String(response.getBody(), StandardCharsets.UTF_8));
            if (parsed instanceof Map) {
                Map<String, Object> data = Json.asObject(parsed);
                Object found = data.containsKey("error") ? data.get("error") : data;
                if (found instanceof String) {
                    message = (String) found;
                } else if (found instanceof Map) {
                    Map<String, Object> error = Json.asObject(found);
                    code = text(error.get("code"));
                    message = text(error.get("message"));
                }
            }
        } catch (IllegalStateException e) {
            // A refusal without a JSON body is still a refusal.
        }
        if (code.isEmpty()) {
            code = "INTERNAL_SERVER_ERROR";
        }
        if (message.isEmpty()) {
            message = reason(response.getStatus());
        }
        String retryAfter = response.getHeaders().getOrDefault("retry-after", "").trim();
        Integer seconds = null;
        if (!retryAfter.isEmpty() && retryAfter.length() < 10 && retryAfter.chars().allMatch(Character::isDigit)) {
            seconds = Integer.parseInt(retryAfter);
        }
        return new MeengException(response.getStatus(), code, message, seconds);
    }

    private static String text(Object value) {
        return value instanceof String ? (String) value : "";
    }

    private static String reason(int status) {
        switch (status) {
            case 400:
                return "Bad Request";
            case 401:
                return "Unauthorized";
            case 403:
                return "Forbidden";
            case 404:
                return "Not Found";
            case 409:
                return "Conflict";
            case 422:
                return "Unprocessable Entity";
            case 429:
                return "Too Many Requests";
            case 500:
                return "Internal Server Error";
            case 502:
                return "Bad Gateway";
            case 503:
                return "Service Unavailable";
            default:
                return "Error";
        }
    }
}
