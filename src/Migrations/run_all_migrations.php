<?php

require_once './vendor/autoload.php';

use Illuminate\Database\Capsule\Manager;
use Termorize\Migrations\PendingTaskMigration;
use Termorize\Migrations\QuestionMigration;
use Termorize\Migrations\TelegramMigration;
use Termorize\Migrations\TranslateModelMigration;
use Termorize\Migrations\UserSettingMigration;
use Termorize\Migrations\VocabularyItemMigration;
use Termorize\Services\Logger;

$kernel = new Termorize\Services\Kernel();
$kernel->connectDatabase();

const MIGRATIONS_CLASSES = [
    TelegramMigration::class,
    TranslateModelMigration::class,
    VocabularyItemMigration::class,
    UserSettingMigration::class,
    PendingTaskMigration::class,
    QuestionMigration::class,
];

foreach (MIGRATIONS_CLASSES as $migrationClass) {
    $migration = new $migrationClass();
    if (!Manager::schema()->hasTable($migration->getTable())) {
        $migration->migrate();

        Logger::info("$migrationClass executed!");
    }
}

Logger::info('Migrations processed');
