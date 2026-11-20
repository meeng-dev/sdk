<?php

// The router `php -S` runs for the test of the real HTTP: it answers what it
// was sent, in the shapes the API uses.

declare(strict_types=1);

header('Content-Type: application/json');
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
if ($path === '/api/v1/emails/redirect/cancel') {
    http_response_code(302);
    header('Location: https://elsewhere.test/');
    return;
}
if (($_SERVER['HTTP_AUTHORIZATION'] ?? '') !== 'Bearer mg_test_key') {
    http_response_code(403);
    header('Retry-After: 3');
    echo json_encode(['error' => ['code' => 'FORBIDDEN', 'message' => 'Invalid API token']]);
    return;
}
echo json_encode([
    'emailId' => json_decode((string) file_get_contents('php://input'), true)['subject'] ?? '',
    'idempotency' => $_SERVER['HTTP_IDEMPOTENCY_KEY'] ?? '',
    'where' => $_SERVER['REQUEST_METHOD'] . ' ' . $_SERVER['REQUEST_URI'],
]);
