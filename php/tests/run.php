<?php

// meeng's PHP SDK tests, with no dependency to install: php tests/run.php

declare(strict_types=1);

use Meeng\Meeng;
use Meeng\MeengException;
use Meeng\Webhooks;
use Meeng\WebhookVerificationException;

spl_autoload_register(static function (string $class): void {
    if (str_starts_with($class, 'Meeng\\')) {
        require __DIR__ . '/../src/' . str_replace('\\', '/', substr($class, 6)) . '.php';
    }
});

$tests = [];
$failures = 0;

function check(bool $condition, string $what): void
{
    if (!$condition) {
        throw new \RuntimeException($what);
    }
}

/**
 * @param callable(array{method: string, url: string, headers: array<string, string>, body: ?string}): array{0: int, 1: mixed} $answer
 * @return array{0: Meeng, 1: \ArrayObject<int, array{method: string, url: string, headers: array<string, string>, body: ?string}>}
 */
function client(callable $answer): array
{
    $seen = new \ArrayObject();
    $transport = static function (string $method, string $url, array $headers, ?string $body, float $timeout) use ($answer, $seen): array {
        $request = ['method' => $method, 'url' => $url, 'headers' => $headers, 'body' => $body];
        $seen[] = $request;
        [$status, $payload] = $answer($request);
        return [$status, ['retry-after' => '7'], is_string($payload) ? $payload : json_encode($payload)];
    };
    return [new Meeng('mg_test_key', 'https://meeng.test/', transport: $transport), $seen];
}

$tests['sending an email carries the key, the body and the idempotency key'] = static function (): void {
    [$meeng, $seen] = client(static fn (): array => [200, ['emailId' => 'em_1']]);
    $sent = $meeng->emails->send(['from' => 'hello@acme.com', 'to' => ['a@x.com', 'b@x.com'], 'subject' => 'Hi', 'html' => '<p>Hi</p>'], 'order-1');
    check($sent['emailId'] === 'em_1', 'answer');
    $request = $seen[0];
    check($request['method'] === 'POST' && $request['url'] === 'https://meeng.test/api/v1/emails', 'request ' . $request['url']);
    check($request['headers']['Authorization'] === 'Bearer mg_test_key', 'key');
    check($request['headers']['Idempotency-Key'] === 'order-1', 'idempotency key');
    check($request['headers']['Content-Type'] === 'application/json', 'content type');
    check($request['headers']['User-Agent'] === 'meeng-php/' . \Meeng\Version::SDK, 'user agent');
    check(json_decode((string) $request['body'], true) === ['from' => 'hello@acme.com', 'to' => ['a@x.com', 'b@x.com'], 'subject' => 'Hi', 'html' => '<p>Hi</p>'], 'body ' . $request['body']);
};

$tests['a write with no body of its own sends an empty object'] = static function (): void {
    [$meeng, $seen] = client(static fn (): array => [200, ['emailId' => 'em_1']]);
    $meeng->emails->cancel('em_1');
    check($seen[0]['method'] === 'POST' && $seen[0]['url'] === 'https://meeng.test/api/v1/emails/em_1/cancel' && $seen[0]['body'] === '{}', 'request ' . $seen[0]['body']);
};

$tests['a send without an idempotency key sends none'] = static function (): void {
    [$meeng, $seen] = client(static fn (): array => [200, ['emailId' => 'em_1']]);
    $meeng->emails->send(['from' => 'hello@acme.com', 'to' => 'a@x.com', 'subject' => 'Hi', 'text' => 'Hi']);
    check(!array_key_exists('Idempotency-Key', $seen[0]['headers']), 'an Idempotency-Key was sent');
};

$tests['a batch answers the list of ids'] = static function (): void {
    [$meeng, $seen] = client(static fn (): array => [200, ['data' => [['emailId' => 'em_1'], ['emailId' => 'em_2']]]]);
    $got = $meeng->emails->batch([
        ['from' => 'hello@acme.com', 'to' => 'a@x.com', 'subject' => 'One', 'text' => '1'],
        ['from' => 'hello@acme.com', 'to' => 'b@x.com', 'subject' => 'Two', 'text' => '2'],
    ]);
    check($got['data'][1]['emailId'] === 'em_2', 'answer');
    check($seen[0]['url'] === 'https://meeng.test/api/v1/emails/batch' && count(json_decode((string) $seen[0]['body'], true)) === 2, 'request');
};

$tests['listing emails sends the filters, a list once for each value'] = static function (): void {
    [$meeng, $seen] = client(static fn (): array => [200, ['data' => [], 'count' => 0]]);
    $got = $meeng->emails->list(['page' => 2, 'limit' => 10, 'startDate' => '2026-11-01T00:00:00Z', 'domainId' => ['1', '2']]);
    check($got['count'] === 0, 'answer');
    check($seen[0]['method'] === 'GET', 'method');
    check($seen[0]['url'] === 'https://meeng.test/api/v1/emails?page=2&limit=10&startDate=2026-11-01T00%3A00%3A00Z&domainId=1&domainId=2', 'url ' . $seen[0]['url']);
};

$tests['a path parameter is escaped and a number is accepted'] = static function (): void {
    [$meeng, $seen] = client(static fn (): array => [200, ['id' => 7]]);
    $meeng->domains->get(7);
    $meeng->emails->get('a/b c');
    check($seen[0]['url'] === 'https://meeng.test/api/v1/domains/7', 'domain ' . $seen[0]['url']);
    check($seen[1]['url'] === 'https://meeng.test/api/v1/emails/a%2Fb%20c', 'email ' . $seen[1]['url']);
};

$tests['a list the API answers without an object around it is a list'] = static function (): void {
    [$meeng] = client(static fn (): array => [200, [['id' => 'cb_1', 'name' => 'News']]]);
    $books = $meeng->contactBooks->list();
    check(count($books) === 1 && $books[0]['name'] === 'News', 'books');
};

$tests['a delete can carry a body'] = static function (): void {
    [$meeng, $seen] = client(static fn (): array => [200, ['success' => true, 'count' => 2]]);
    $got = $meeng->contacts->bulkDelete('cb_1', ['contactIds' => ['c1', 'c2']]);
    check($got['count'] === 2, 'answer');
    check($seen[0]['method'] === 'DELETE' && $seen[0]['url'] === 'https://meeng.test/api/v1/contactBooks/cb_1/contacts/bulk' && $seen[0]['body'] === '{"contactIds":["c1","c2"]}', 'request ' . $seen[0]['body']);
};

$tests['upsert takes the contact\'s id in the path'] = static function (): void {
    [$meeng, $seen] = client(static fn (): array => [200, ['contactId' => 'c_1']]);
    $got = $meeng->contacts->upsert('cb_1', 'c_1', ['email' => 'a@x.com', 'firstName' => 'Ana']);
    check($got['contactId'] === 'c_1' && $seen[0]['method'] === 'PUT' && $seen[0]['url'] === 'https://meeng.test/api/v1/contactBooks/cb_1/contacts/c_1', 'request');
};

$tests['a refusal is an exception, in every shape the API writes it'] = static function (): void {
    $cases = [
        [409, ['error' => ['code' => 'NOT_UNIQUE', 'message' => 'Idempotency-Key already used']], 'NOT_UNIQUE', 'Idempotency-Key already used'],
        [403, ['error' => 'Forbidden'], 'INTERNAL_SERVER_ERROR', 'Forbidden'],
        [400, ['code' => 'BAD_REQUEST', 'message' => 'bad'], 'BAD_REQUEST', 'bad'],
        [400, ['success' => false, 'error' => ['issues' => [['code' => 'invalid_type', 'path' => ['from'], 'message' => 'Required']], 'name' => 'ZodError']], 'ZodError', 'from: Required'],
        [502, '<html>', 'INTERNAL_SERVER_ERROR', 'Bad Gateway'],
    ];
    foreach ($cases as [$status, $body, $code, $message]) {
        [$meeng] = client(static fn (): array => [$status, $body]);
        try {
            $meeng->emails->cancel('em_1');
            check(false, "a {$status} did not throw");
        } catch (MeengException $refused) {
            check($refused->status === $status && $refused->errorCode === $code && $refused->retryAfter === 7, "refusal {$status}");
            check($refused->getMessage() === "meeng: {$status} {$code}: {$message}", 'message ' . $refused->getMessage());
        }
    }
};

$tests['an email\'s recipients may come as one string or a list'] = static function (): void {
    [$meeng] = client(static fn (): array => [200, ['id' => 'em_1', 'to' => 'a@x.com']]);
    check($meeng->emails->get('em_1')['to'] === 'a@x.com', 'to');
};

$tests['the key and the address come from the environment'] = static function (): void {
    putenv('MEENG_API_KEY=mg_env');
    putenv('MEENG_BASE_URL=http://127.0.0.1:1');
    try {
        $seen = null;
        $meeng = new Meeng(transport: static function (string $method, string $url, array $headers) use (&$seen): array {
            $seen = [$url, $headers['Authorization']];
            return [200, [], '{}'];
        });
        $meeng->emails->get('em_1');
        check($seen === ['http://127.0.0.1:1/api/v1/emails/em_1', 'Bearer mg_env'], 'environment ' . json_encode($seen));
        putenv('MEENG_API_KEY');
        try {
            new Meeng();
            check(false, 'a missing key did not throw');
        } catch (\InvalidArgumentException) {
        }
    } finally {
        putenv('MEENG_API_KEY');
        putenv('MEENG_BASE_URL');
    }
};

$tests['the address must be https, except to localhost'] = static function (): void {
    foreach (['http://app.meeng.test', 'ftp://app.meeng.test', 'app.meeng.test', 'http://localhost.evil.test'] as $base) {
        try {
            new Meeng('k', $base);
            check(false, "{$base} was accepted");
        } catch (\InvalidArgumentException) {
        }
    }
    foreach (['https://app.meeng.test', 'http://localhost:3000', 'http://127.0.0.1:3000', 'http://[::1]:3000'] as $base) {
        new Meeng('k', $base);
    }
};

const WEBHOOK_SECRET = 'whsec_test';
const WEBHOOK_BODY = '{"id":"evt_1","type":"email.delivered","createdAt":"2026-11-20T12:00:00Z","data":{"id":"em_1","status":"DELIVERED","from":"h@acme.com","to":["a@x.com"],"occurredAt":"2026-11-20T12:00:00Z"}}';

function sign(string $timestamp, string $body = WEBHOOK_BODY, string $secret = WEBHOOK_SECRET): string
{
    return 'v1=' . hash_hmac('sha256', $timestamp . '.' . $body, $secret);
}

$tests['a signed webhook is verified and read as an event'] = static function (): void {
    $timestamp = (string) (int) floor(microtime(true) * 1000);
    $webhooks = (new Meeng('k', 'https://meeng.test'))->webhooks(WEBHOOK_SECRET);
    $headers = ['X-Meeng-Signature' => sign($timestamp), 'X-Meeng-Timestamp' => $timestamp];
    check($webhooks->verify(WEBHOOK_BODY, $headers), 'verify');
    $event = $webhooks->constructEvent(WEBHOOK_BODY, $headers);
    check($event['type'] === 'email.delivered' && $event['data']['to'] === ['a@x.com'], 'event');
};

$tests['the useSend headers are accepted as well, in any case'] = static function (): void {
    $timestamp = (string) (int) floor(microtime(true) * 1000);
    $headers = ['x-usesend-signature' => sign($timestamp), 'x-usesend-timestamp' => $timestamp];
    check((new Webhooks(WEBHOOK_SECRET))->verify(WEBHOOK_BODY, $headers), 'legacy headers');
};

$tests['a webhook is refused with the code that says why'] = static function (): void {
    $now = 1_800_000_000_000;
    $timestamp = (string) $now;
    $old = (string) ($now - 6 * 60 * 1000);
    $webhooks = new Webhooks(WEBHOOK_SECRET);
    $good = ['X-Meeng-Signature' => sign($timestamp), 'X-Meeng-Timestamp' => $timestamp];
    $stale = ['X-Meeng-Signature' => sign($old), 'X-Meeng-Timestamp' => $old];
    $refused = static function (array $headers, string $body = WEBHOOK_BODY, ?string $secret = null, ?int $tolerance = null) use ($webhooks, $now): string {
        try {
            $webhooks->constructEvent($body, $headers, $secret, $tolerance, $now);
        } catch (WebhookVerificationException $error) {
            return $error->errorCode;
        }
        return 'ACCEPTED';
    };
    $cases = [
        'ACCEPTED' => $refused($good),
        'MISSING_SIGNATURE' => $refused(['X-Meeng-Timestamp' => $timestamp]),
        'MISSING_TIMESTAMP' => $refused(['X-Meeng-Signature' => sign($timestamp)]),
        'INVALID_SIGNATURE_FORMAT' => $refused(['X-Meeng-Signature' => substr(sign($timestamp), 3), 'X-Meeng-Timestamp' => $timestamp]),
        'INVALID_TIMESTAMP' => $refused(['X-Meeng-Signature' => sign($timestamp), 'X-Meeng-Timestamp' => 'yesterday']),
        'TIMESTAMP_OUT_OF_RANGE' => $refused($stale),
        'SIGNATURE_MISMATCH' => $refused($good, WEBHOOK_BODY, 'another'),
        'INVALID_JSON' => $refused(['X-Meeng-Signature' => sign($timestamp, 'not json'), 'X-Meeng-Timestamp' => $timestamp], 'not json'),
    ];
    foreach ($cases as $want => $got) {
        check($want === $got, "answered {$got}, wanted {$want}");
    }
    check($refused($stale, tolerance: -1) === 'ACCEPTED', 'tolerance off');
    check($refused($stale, tolerance: 10 * 60 * 1000) === 'ACCEPTED', 'tolerance wide');
    check($refused($good, str_replace('em_1', 'em_2', WEBHOOK_BODY)) === 'SIGNATURE_MISMATCH', 'another body');
};

$tests['the real HTTP sends and reads through curl and never follows a redirect'] = static function (): void {
    $socket = stream_socket_server('tcp://127.0.0.1:0');
    $port = (int) substr(strrchr((string) stream_socket_get_name($socket, false), ':'), 1);
    fclose($socket);
    $server = proc_open([PHP_BINARY, '-S', "127.0.0.1:{$port}", __DIR__ . '/server.php'], [1 => ['file', '/dev/null', 'w'], 2 => ['file', '/dev/null', 'w']], $pipes);
    try {
        for ($tries = 0; $tries < 50 && @fsockopen('127.0.0.1', $port) === false; $tries++) {
            usleep(100_000);
        }
        $meeng = new Meeng('mg_test_key', "http://127.0.0.1:{$port}");
        $sent = $meeng->emails->send(['from' => 'a@x.com', 'to' => 'b@x.com', 'subject' => 'Curso'], 'create-1');
        check($sent['emailId'] === 'Curso' && $sent['idempotency'] === 'create-1' && $sent['where'] === 'POST /api/v1/emails', 'over HTTP: ' . json_encode($sent));
        try {
            (new Meeng('mg_test_other', "http://127.0.0.1:{$port}"))->emails->get('em_1');
            check(false, 'a 403 did not throw');
        } catch (MeengException $refused) {
            check($refused->status === 403 && $refused->errorCode === 'FORBIDDEN' && $refused->retryAfter === 3, 'over HTTP refusal');
        }
        try {
            $meeng->emails->cancel('redirect');
            check(false, 'a redirect did not throw');
        } catch (MeengException $refused) {
            check($refused->status === 302, 'a redirect was followed or hidden');
        }
    } finally {
        proc_terminate($server);
        proc_close($server);
    }
};

foreach ($tests as $name => $test) {
    try {
        $test();
        echo "ok   {$name}\n";
    } catch (\Throwable $error) {
        $failures++;
        echo "FAIL {$name}: {$error->getMessage()}\n";
    }
}
echo $failures === 0 ? "\nPASS " . count($tests) . " tests\n" : "\nFAIL {$failures} of " . count($tests) . " tests\n";
exit($failures === 0 ? 0 : 1);
