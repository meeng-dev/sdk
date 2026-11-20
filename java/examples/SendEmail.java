import dev.meeng.EmailsSendResponse;
import dev.meeng.EmailsService;
import dev.meeng.Meeng;
import dev.meeng.MeengException;
import dev.meeng.RequestOptions;
import java.util.List;

/**
 * Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client at
 * another app, the address from MEENG_BASE_URL.
 *
 * <pre>
 * mvn -q package -DskipTests
 * MEENG_API_KEY=mg_xxx java -cp target/classes examples/SendEmail.java
 * </pre>
 */
public class SendEmail {
    public static void main(String[] args) {
        Meeng meeng = new Meeng();
        try {
            EmailsSendResponse sent = meeng.emails().send(
                    EmailsService.SendParams.builder()
                            .from("Acme <contato@acme.com.br>")
                            .to(List.of("ana@example.com"))
                            .subject("Bem-vinda à Acme")
                            .html("<p>Olá, Ana! Sua conta está pronta.</p>")
                            .text("Olá, Ana! Sua conta está pronta.")
                            .build(),
                    // A key makes the send safe to repeat: the same key with the same body
                    // answers the first result again, and nothing is sent twice.
                    RequestOptions.builder().idempotencyKey("boas-vindas-ana-2026-11-20").build());
            System.out.println("e-mail na fila: " + sent.getEmailId());
        } catch (MeengException error) {
            System.err.println(error.getMessage());
            System.exit(1);
        }
    }
}
