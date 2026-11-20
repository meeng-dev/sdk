<?php

declare(strict_types=1);

namespace Meeng;

/** What a service calls the API through. */
interface Requester
{
    /**
     * @param array<string, mixed> $query
     * @param array<mixed>|null $body
     * @return mixed the answer's JSON, decoded
     */
    public function request(string $method, string $path, array $query = [], ?array $body = null, ?string $idempotencyKey = null): mixed;
}
