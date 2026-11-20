<?php

// Loads the SDK from src/ so that the examples run without Composer. In your own
// project Composer's autoloader does this: require 'vendor/autoload.php'.

declare(strict_types=1);

spl_autoload_register(static function (string $class): void {
    if (str_starts_with($class, 'Meeng\\')) {
        require __DIR__ . '/../src/' . str_replace('\\', '/', substr($class, 6)) . '.php';
    }
});
