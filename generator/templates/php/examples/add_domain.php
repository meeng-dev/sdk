<?php

// Adds a sending domain, prints the DNS records to publish and asks for them to
// be checked.
//
//   MEENG_API_KEY=mg_xxx php examples/add_domain.php

declare(strict_types=1);

require __DIR__ . '/autoload.php';

use Meeng\Meeng;
use Meeng\MeengException;

$meeng = new Meeng();

try {
    $domain = $meeng->domains->create(['name' => 'acme.com.br', 'region' => 'us-east-1']);
    echo 'publique estes registros no DNS de ', $domain['name'], "\n";
    foreach ($domain['dnsRecords'] as $record) {
        printf("  %-5s %-40s %s\n", $record['type'], $record['name'], $record['value']);
    }

    $meeng->domains->verify($domain['id']);
    $checked = $meeng->domains->get($domain['id']);
    echo 'situação: ', $checked['status'], "\n";
} catch (MeengException $error) {
    fwrite(STDERR, $error->getMessage() . "\n");
    exit(1);
}
