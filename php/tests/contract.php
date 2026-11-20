<?php

// The contract test calls every method of the SDK against a fake of the API that
// is built from the API's contract and refuses what the contract does not allow.
// check.sh starts the fake and sets MEENG_FAKE_URL. Run by tests/run.php.

declare(strict_types=1);

use Meeng\Meeng;
use Meeng\MeengException;

$fake = getenv('MEENG_FAKE_URL');
$contract = static function (callable $test) use ($fake): callable {
    return static function () use ($test, $fake): void {
        if (!is_string($fake) || $fake === '') {
            echo "     (skipped: MEENG_FAKE_URL is not set, start the fake API with go run ./cmd/fakeapi)\n";
            return;
        }
        $test(new Meeng('mg_contract_test', $fake, retryDelay: 0.001));
    };
};

$tests['contract: every method makes the call the contract describes'] = $contract(static function (Meeng $meeng): void {
    try {
        $meeng->emails->send(['to' => ['hello@example.com'], 'from' => 'hello@example.com'], 'contract-emails-send');
    } catch (MeengException $e) {
        check(false, 'emails.send' . ': ' . $e->getMessage());
    }
    try {
        $meeng->emails->batch([['to' => ['hello@example.com'], 'from' => 'hello@example.com']], 'contract-emails-batch');
    } catch (MeengException $e) {
        check(false, 'emails.batch' . ': ' . $e->getMessage());
    }
    try {
        $meeng->emails->get('id_emailid');
    } catch (MeengException $e) {
        check(false, 'emails.get' . ': ' . $e->getMessage());
    }
    try {
        $meeng->emails->list(['page' => 2, 'limit' => 10, 'startDate' => '2026-11-01T00:00:00Z', 'endDate' => '2026-11-30T00:00:00Z', 'domainId' => ['1', '2']]);
    } catch (MeengException $e) {
        check(false, 'emails.list' . ': ' . $e->getMessage());
    }
    try {
        $meeng->emails->update('id_emailid', ['scheduledAt' => '2026-12-01T09:00:00Z']);
    } catch (MeengException $e) {
        check(false, 'emails.update' . ': ' . $e->getMessage());
    }
    try {
        $meeng->emails->cancel('id_emailid');
    } catch (MeengException $e) {
        check(false, 'emails.cancel' . ': ' . $e->getMessage());
    }
    try {
        $meeng->domains->list();
    } catch (MeengException $e) {
        check(false, 'domains.list' . ': ' . $e->getMessage());
    }
    try {
        $meeng->domains->create(['name' => 'example.com', 'region' => 'us-east-1']);
    } catch (MeengException $e) {
        check(false, 'domains.create' . ': ' . $e->getMessage());
    }
    try {
        $meeng->domains->verify('id_id');
    } catch (MeengException $e) {
        check(false, 'domains.verify' . ': ' . $e->getMessage());
    }
    try {
        $meeng->domains->get('id_id');
    } catch (MeengException $e) {
        check(false, 'domains.get' . ': ' . $e->getMessage());
    }
    try {
        $meeng->domains->delete('id_id');
    } catch (MeengException $e) {
        check(false, 'domains.delete' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contacts->create('id_contactbookid', ['email' => 'hello@example.com']);
    } catch (MeengException $e) {
        check(false, 'contacts.create' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contacts->list('id_contactbookid', ['emails' => 'a@example.com', 'page' => 2, 'limit' => 10, 'ids' => 'c1,c2']);
    } catch (MeengException $e) {
        check(false, 'contacts.list' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contacts->get('id_contactbookid', 'id_contactid');
    } catch (MeengException $e) {
        check(false, 'contacts.get' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contacts->update('id_contactbookid', 'id_contactid', []);
    } catch (MeengException $e) {
        check(false, 'contacts.update' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contacts->upsert('id_contactbookid', 'id_contactid', ['email' => 'hello@example.com']);
    } catch (MeengException $e) {
        check(false, 'contacts.upsert' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contacts->delete('id_contactbookid', 'id_contactid');
    } catch (MeengException $e) {
        check(false, 'contacts.delete' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contacts->bulkCreate('id_contactbookid', [['email' => 'hello@example.com']]);
    } catch (MeengException $e) {
        check(false, 'contacts.bulkCreate' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contacts->bulkDelete('id_contactbookid', ['contactIds' => ['c1', 'c2']]);
    } catch (MeengException $e) {
        check(false, 'contacts.bulkDelete' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contactBooks->list();
    } catch (MeengException $e) {
        check(false, 'contactBooks.list' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contactBooks->create(['name' => 'example.com']);
    } catch (MeengException $e) {
        check(false, 'contactBooks.create' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contactBooks->get('id_contactbookid');
    } catch (MeengException $e) {
        check(false, 'contactBooks.get' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contactBooks->update('id_contactbookid', []);
    } catch (MeengException $e) {
        check(false, 'contactBooks.update' . ': ' . $e->getMessage());
    }
    try {
        $meeng->contactBooks->delete('id_contactbookid');
    } catch (MeengException $e) {
        check(false, 'contactBooks.delete' . ': ' . $e->getMessage());
    }
    try {
        $meeng->campaigns->create(['name' => 'example.com', 'from' => 'hello@example.com', 'subject' => 'sample', 'contactBookId' => 'id_contactbookid']);
    } catch (MeengException $e) {
        check(false, 'campaigns.create' . ': ' . $e->getMessage());
    }
    try {
        $meeng->campaigns->list(['page' => 2, 'status' => 'DRAFT', 'search' => 'news']);
    } catch (MeengException $e) {
        check(false, 'campaigns.list' . ': ' . $e->getMessage());
    }
    try {
        $meeng->campaigns->get('id_campaignid');
    } catch (MeengException $e) {
        check(false, 'campaigns.get' . ': ' . $e->getMessage());
    }
    try {
        $meeng->campaigns->delete('id_campaignid');
    } catch (MeengException $e) {
        check(false, 'campaigns.delete' . ': ' . $e->getMessage());
    }
    try {
        $meeng->campaigns->schedule('id_campaignid', []);
    } catch (MeengException $e) {
        check(false, 'campaigns.schedule' . ': ' . $e->getMessage());
    }
    try {
        $meeng->campaigns->pause('id_campaignid');
    } catch (MeengException $e) {
        check(false, 'campaigns.pause' . ': ' . $e->getMessage());
    }
    try {
        $meeng->campaigns->resume('id_campaignid');
    } catch (MeengException $e) {
        check(false, 'campaigns.resume' . ': ' . $e->getMessage());
    }
    try {
        $meeng->analytics->emailTimeSeries(['days' => '7', 'domainId' => '1']);
    } catch (MeengException $e) {
        check(false, 'analytics.emailTimeSeries' . ': ' . $e->getMessage());
    }
    try {
        $meeng->analytics->reputationMetrics(['domainId' => '1']);
    } catch (MeengException $e) {
        check(false, 'analytics.reputationMetrics' . ': ' . $e->getMessage());
    }
});

$tests['contract: a refusal is a typed error'] = $contract(static function (Meeng $meeng): void {
    try {
        $meeng->emails->get('fake-404');
        check(false, 'a 404 was accepted');
    } catch (MeengException $e) {
        check($e->status === 404 && $e->errorCode === 'NOT_FOUND', 'a 404 is ' . $e->getMessage());
    }
});

$tests['contract: a too many requests answer is repeated until it passes'] = $contract(static function (Meeng $meeng): void {
    $meeng->domains->get('flaky-429-2');
    try {
        $meeng->domains->get('fake-429');
        check(false, 'a 429 that never passes was accepted');
    } catch (MeengException $e) {
        check($e->status === 429 && $e->errorCode === 'RATE_LIMITED', 'a 429 is ' . $e->getMessage());
    }
});

$tests['contract: a server failure is repeated for a read'] = $contract(static function (Meeng $meeng): void {
    $meeng->emails->get('flaky-503-2');
});

$tests['contract: the same idempotency key answers the same email'] = $contract(static function (Meeng $meeng): void {
    $key = 'order-' . bin2hex(random_bytes(6));
    $email = ['from' => 'hello@example.com', 'to' => ['a@example.com']];
    $first = $meeng->emails->send($email, $key);
    $again = $meeng->emails->send($email, $key);
    check($first['emailId'] === $again['emailId'], 'the same key answered another email');
    try {
        $meeng->emails->send(['from' => 'hello@example.com', 'to' => ['b@example.com']], $key);
        check(false, 'another body under the same key was accepted');
    } catch (MeengException $e) {
        check($e->status === 409 && $e->errorCode === 'NOT_UNIQUE', 'a 409 is ' . $e->getMessage());
    }
});

$tests['contract: an email needs its required fields'] = $contract(static function (Meeng $meeng): void {
    try {
        $meeng->emails->send(['to' => 'a@example.com']);
        check(false, 'a send with no from was accepted');
    } catch (MeengException $e) {
        check($e->status === 400 && $e->errorCode === 'ZodError', 'a 400 is ' . $e->getMessage());
    }
});
