import dev.meeng.EmailsBatchResponse;
import dev.meeng.EmailsService;
import dev.meeng.Meeng;
import dev.meeng.MeengException;
import dev.meeng.RequestOptions;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Sends up to 100 e-mails in one request.
 *
 * <pre>
 * mvn -q package -DskipTests
 * MEENG_API_KEY=mg_xxx java -cp target/classes examples/SendBatch.java
 * </pre>
 */
public class SendBatch {
    public static void main(String[] args) {
        Meeng meeng = new Meeng();
        Map<String, String> people = Map.of("Ana", "ana@example.com", "Bruno", "bruno@example.com", "Carla", "carla@example.com");
        List<EmailsService.BatchParams> batch = new ArrayList<>();
        people.forEach((name, email) -> batch.add(EmailsService.BatchParams.builder()
                .from("Acme <contato@acme.com.br>")
                .to(List.of(email))
                .subject("Seu extrato de novembro")
                .html("<p>Olá, " + name + "! Seu extrato está disponível.</p>")
                .build()));
        try {
            EmailsBatchResponse sent = meeng.emails().batch(batch, RequestOptions.builder().idempotencyKey("extrato-2026-11").build());
            for (int i = 0; i < sent.getData().size(); i++) {
                System.out.println((i + 1) + ": " + sent.getData().get(i).getEmailId());
            }
        } catch (MeengException error) {
            System.err.println(error.getMessage());
            System.exit(1);
        }
    }
}
