<?php

// Schedules an e-mail, moves it to another time and cancels it.
//
//   MEENG_API_KEY=mg_xxx php examples/schedule_email.php

declare(strict_types=1);

require __DIR__ . '/autoload.php';

use Meeng\Meeng;
use Meeng\MeengException;

$meeng = new Meeng();
$inHours = static fn (int $hours): string => gmdate('Y-m-d\TH:i:s\Z', time() + $hours * 3600);

try {
    $sent = $meeng->emails->send([
        'from' => 'Acme <contato@acme.com.br>',
        'to' => 'ana@example.com',
        'subject' => 'Lembrete: sua reunião é amanhã',
        'html' => '<p>Nos vemos às 10h.</p>',
        'scheduledAt' => $inHours(24),
    ]);
    echo 'agendado: ', $sent['emailId'], "\n";

    $meeng->emails->update($sent['emailId'], ['scheduledAt' => $inHours(48)]);
    echo "remarcado\n";

    $meeng->emails->cancel($sent['emailId']);
    echo "cancelado\n";
} catch (MeengException $error) {
    fwrite(STDERR, $error->getMessage() . "\n");
    exit(1);
}
