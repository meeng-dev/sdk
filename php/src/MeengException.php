<?php

declare(strict_types=1);

namespace Meeng;

/**
 * A request the API refused. $errorCode is the API's, such as NOT_UNIQUE,
 * FORBIDDEN or NOT_FOUND; it is INTERNAL_SERVER_ERROR when the answer carried
 * none. $status is the HTTP status; $retryAfter, on a 429, how many seconds to
 * wait.
 */
final class MeengException extends \RuntimeException
{
    public function __construct(
        public readonly int $status,
        public readonly string $errorCode,
        string $message = '',
        public readonly ?int $retryAfter = null,
    ) {
        parent::__construct($message !== '' ? "meeng: {$status} {$errorCode}: {$message}" : "meeng: {$status} {$errorCode}", $status);
    }

    /**
     * The error a refused answer carries: {"error": {"code", "message"}},
     * {"error": "text"} or the error itself.
     *
     * @param array<string, string> $headers in lower case
     */
    public static function fromResponse(int $status, array $headers, string $body): self
    {
        $code = 'INTERNAL_SERVER_ERROR';
        $message = self::reason($status);
        $data = json_decode($body, true);
        if (is_array($data)) {
            $found = $data['error'] ?? $data;
            if (is_string($found)) {
                $found = ['message' => $found];
            }
            if (is_array($found)) {
                if (is_string($found['code'] ?? null) && $found['code'] !== '') {
                    $code = $found['code'];
                } elseif (is_string($found['name'] ?? null) && $found['name'] !== '') {
                    // A refused validation answers {"success": false, "error": {"issues", "name"}}.
                    $code = $found['name'];
                }
                $first = is_array($found['issues'] ?? null) ? ($found['issues'][0] ?? null) : null;
                if (is_string($found['message'] ?? null) && $found['message'] !== '') {
                    $message = $found['message'];
                } elseif (is_array($first) && is_string($first['message'] ?? null)) {
                    $path = is_array($first['path'] ?? null) ? implode('.', $first['path']) : '';
                    $message = ($path !== '' ? $path . ': ' : '') . $first['message'];
                }
            }
        }
        $retryAfter = $headers['retry-after'] ?? '';
        return new self($status, $code, $message, ctype_digit($retryAfter) ? (int) $retryAfter : null);
    }

    private static function reason(int $status): string
    {
        return [
            400 => 'Bad Request', 401 => 'Unauthorized', 403 => 'Forbidden', 404 => 'Not Found', 409 => 'Conflict',
            422 => 'Unprocessable Entity', 429 => 'Too Many Requests', 500 => 'Internal Server Error',
            502 => 'Bad Gateway', 503 => 'Service Unavailable',
        ][$status] ?? 'Error';
    }
}
