import dev.meeng.EmailsSendResponse;
import dev.meeng.EmailsService;
import dev.meeng.Meeng;
import dev.meeng.MeengException;
import java.time.Duration;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.List;

/**
 * Schedules an e-mail, moves it to another time and cancels it.
 *
 * <pre>
 * mvn -q package -DskipTests
 * MEENG_API_KEY=mg_xxx java -cp target/classes examples/ScheduleEmail.java
 * </pre>
 */
public class ScheduleEmail {
    private static String inHours(long hours) {
        return Instant.now().plus(Duration.ofHours(hours)).truncatedTo(ChronoUnit.SECONDS).toString();
    }

    public static void main(String[] args) {
        Meeng meeng = new Meeng();
        try {
            EmailsSendResponse sent = meeng.emails().send(EmailsService.SendParams.builder()
                    .from("Acme <contato@acme.com.br>")
                    .to(List.of("ana@example.com"))
                    .subject("Lembrete: sua reunião é amanhã")
                    .html("<p>Nos vemos às 10h.</p>")
                    .scheduledAt(inHours(24))
                    .build());
            System.out.println("agendado: " + sent.getEmailId());

            meeng.emails().update(sent.getEmailId(), EmailsService.UpdateParams.builder().scheduledAt(inHours(48)).build());
            System.out.println("remarcado");

            meeng.emails().cancel(sent.getEmailId());
            System.out.println("cancelado");
        } catch (MeengException error) {
            System.err.println(error.getMessage());
            System.exit(1);
        }
    }
}
