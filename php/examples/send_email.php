<?php

// Sends one e-mail. The key comes from MEENG_API_KEY and, to point the client at
// another app, the address from MEENG_BASE_URL.
//
//   MEENG_API_KEY=mg_xxx php examples/send_email.php

declare(strict_types=1);

require __DIR__ . '/autoload.php';

use Meeng\Meeng;
use Meeng\MeengException;

$meeng = new Meeng();

try {
    $sent = $meeng->emails->send(
        [
            'from' => 'Acme <contato@acme.com.br>',
            'to' => 'ana@example.com',
            'subject' => 'Bem-vinda à Acme',
            'html' => '<p>Olá, Ana! Sua conta está pronta.</p>',
            'text' => 'Olá, Ana! Sua conta está pronta.',
        ],
        // A key makes the send safe to repeat: the same key with the same body
        // answers the first result again, and nothing is sent twice.
        'boas-vindas-ana-2026-11-20',
    );
} catch (MeengException $error) {
    fwrite(STDERR, $error->getMessage() . "\n");
    exit(1);
}

echo 'e-mail na fila: ', $sent['emailId'], "\n";
