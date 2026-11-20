<?php

// Sends up to 100 e-mails in one request.
//
//   MEENG_API_KEY=mg_xxx php examples/send_batch.php

declare(strict_types=1);

require __DIR__ . '/autoload.php';

use Meeng\Meeng;
use Meeng\MeengException;

$meeng = new Meeng();
$people = ['Ana' => 'ana@example.com', 'Bruno' => 'bruno@example.com', 'Carla' => 'carla@example.com'];

$batch = [];
foreach ($people as $name => $email) {
    $batch[] = [
        'from' => 'Acme <contato@acme.com.br>',
        'to' => $email,
        'subject' => 'Seu extrato de novembro',
        'html' => "<p>Olá, {$name}! Seu extrato está disponível.</p>",
    ];
}

try {
    $sent = $meeng->emails->batch($batch, 'extrato-2026-11');
} catch (MeengException $error) {
    fwrite(STDERR, $error->getMessage() . "\n");
    exit(1);
}

foreach ($sent['data'] as $index => $email) {
    echo $index + 1, ': ', $email['emailId'], "\n";
}
