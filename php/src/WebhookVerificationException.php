<?php

declare(strict_types=1);

namespace Meeng;

/**
 * A request meeng did not sign, signed too long ago, or that is not an event.
 * $errorCode is MISSING_SIGNATURE, MISSING_TIMESTAMP, INVALID_SIGNATURE_FORMAT,
 * INVALID_TIMESTAMP, TIMESTAMP_OUT_OF_RANGE, SIGNATURE_MISMATCH, INVALID_BODY
 * or INVALID_JSON.
 */
final class WebhookVerificationException extends \RuntimeException
{
    public function __construct(public readonly string $errorCode, string $message)
    {
        parent::__construct("{$errorCode}: {$message}");
    }
}
