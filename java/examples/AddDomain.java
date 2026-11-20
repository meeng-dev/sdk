import dev.meeng.Domain;
import dev.meeng.DomainDnsRecord;
import dev.meeng.DomainsService;
import dev.meeng.Meeng;
import dev.meeng.MeengException;

/**
 * Adds a sending domain, prints the DNS records to publish and asks for them to
 * be checked.
 *
 * <pre>
 * mvn -q package -DskipTests
 * MEENG_API_KEY=mg_xxx java -cp target/classes examples/AddDomain.java
 * </pre>
 */
public class AddDomain {
    public static void main(String[] args) {
        Meeng meeng = new Meeng();
        try {
            Domain domain = meeng.domains().create(DomainsService.CreateParams.builder().name("acme.com.br").region("us-east-1").build());
            System.out.println("publique estes registros no DNS de " + domain.getName());
            for (DomainDnsRecord record : domain.getDnsRecords()) {
                System.out.printf("  %-5s %-40s %s%n", record.getType(), record.getName(), record.getValue());
            }

            String id = Long.toString(domain.getId());
            meeng.domains().verify(id);
            System.out.println("situação: " + meeng.domains().get(id).getStatus());
        } catch (MeengException error) {
            System.err.println(error.getMessage());
            System.exit(1);
        }
    }
}
