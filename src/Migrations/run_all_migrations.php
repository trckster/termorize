<?php

require_once './vendor/autoload.php';

use Termorize\Migrations\PendingTaskMigration;
use Termorize\Migrations\TelegramMigration;
use Termorize\Migrations\TranslateModelMigration;
use Termorize\Migrations\UserSettingMigration;
use Termorize\Migrations\VocabularyItemMigration;

$kernel = new Termorize\Services\Kernel();
$kernel->connectDatabase();

TelegramMigration::migrate();
TranslateModelMigration::migrate();
VocabularyItemMigration::migrate();
UserSettingMigration::migrate();
PendingTaskMigration::migrate();
