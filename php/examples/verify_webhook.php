<?php

// Checks a webhook the way an endpoint does: the signature and the time are
// verified against the raw body, and then the event is read.
// The request here is made in the program; a real endpoint reads the body and the
// headers of the request it receives, and the secret from its settings.
//
//   php examples/verify_webhook.php

declare(strict_types=1);

require __DIR__ . '/autoload.php';

use Meeng\Webhooks;
use Meeng\WebhookVerificationException;

$secret = 'whsec_exemplo';
$body = json_encode([
    'id' => 'call_1',
    'type' => 'email.delivered',
    'version' => '2026-01-18',
    'createdAt' => '2026-11-20T12:00:00Z',
    'teamId' => 7,
    'attempt' => 1,
    'data' => ['id' => 'em_1', 'status' => 'DELIVERED', 'from' => 'contato@acme.com.br', 'to' => ['ana@example.com'], 'occurredAt' => '2026-11-20T12:00:00Z'],
], JSON_THROW_ON_ERROR);
$timestamp = (string) (int) floor(microtime(true) * 1000);
$headers = [
    'x-meeng-signature' => 'v1=' . hash_hmac('sha256', $timestamp . '.' . $body, $secret),
    'x-meeng-timestamp' => $timestamp,
];

try {
    $event = (new Webhooks($secret))->constructEvent($body, $headers);
} catch (WebhookVerificationException $error) {
    fwrite(STDERR, $error->getMessage() . "\n");
    exit(1);
}

if ($event['type'] === 'email.delivered') {
    echo 'e-mail ', $event['data']['id'], ' entregue a ', implode(', ', $event['data']['to']), "\n";
} else {
    echo 'evento ', $event['type'], "\n";
}
