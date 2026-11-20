import dev.meeng.EmailWebhookEvent;
import dev.meeng.Meeng;
import dev.meeng.WebhookEvent;
import dev.meeng.WebhookVerificationException;
import java.nio.charset.StandardCharsets;
import java.util.HexFormat;
import java.util.Map;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

/**
 * Checks a webhook the way an endpoint does: the signature and the time are
 * verified against the raw body, and then the event is read as a typed value.
 * The request here is made in the program; a real endpoint reads the body and the
 * headers of the request it receives, and the secret from its settings.
 *
 * <pre>
 * mvn -q package -DskipTests
 * java -cp target/classes examples/VerifyWebhook.java
 * </pre>
 */
public class VerifyWebhook {
    public static void main(String[] args) throws Exception {
        String secret = "whsec_exemplo";
        String body = "{\"id\":\"call_1\",\"type\":\"email.delivered\",\"version\":\"2026-01-18\",\"createdAt\":\"2026-11-20T12:00:00Z\","
                + "\"teamId\":7,\"attempt\":1,\"data\":{\"id\":\"em_1\",\"status\":\"DELIVERED\",\"from\":\"contato@acme.com.br\","
                + "\"to\":[\"ana@example.com\"],\"occurredAt\":\"2026-11-20T12:00:00Z\"}}";

        String timestamp = Long.toString(System.currentTimeMillis());
        Mac mac = Mac.getInstance("HmacSHA256");
        mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
        String signature = HexFormat.of().formatHex(mac.doFinal((timestamp + "." + body).getBytes(StandardCharsets.UTF_8)));
        Map<String, String> headers = Map.of("X-Meeng-Signature", "v1=" + signature, "X-Meeng-Timestamp", timestamp);

        try {
            WebhookEvent event = new Meeng("mg_exemplo").webhooks(secret).constructEvent(body, headers);
            if (event instanceof EmailWebhookEvent email) {
                System.out.println("e-mail " + email.getData().getId() + " entregue a " + String.join(", ", email.getData().getTo()));
            } else {
                System.out.println("evento " + event.getType());
            }
        } catch (WebhookVerificationException error) {
            System.err.println(error.getCode() + ": " + error.getMessage());
            System.exit(1);
        }
    }
}
