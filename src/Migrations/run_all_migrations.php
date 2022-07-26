<?php

require_once getBasePath('vendor/autoload.php');

use Termorize\Migrations\TelegramMigration;
use Termorize\Migrations\TranslateModelMigration;

$kernel = new Termorize\Services\Kernel();
$kernel->connectDatabase();

TelegramMigration::migrate();
TranslateModelMigration::migrate();
