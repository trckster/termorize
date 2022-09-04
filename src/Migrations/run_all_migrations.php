<?php

$kernel = new Termorize\Services\Kernel;
$kernel->connectDatabase();

Termorize\Migrations\TelegramMigration::migrate();