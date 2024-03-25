<?php

require_once './vendor/autoload.php';

use Illuminate\Database\Capsule\Manager;
use Termorize\Migrations\PendingTaskMigration;
use Termorize\Migrations\TelegramMigration;
use Termorize\Migrations\TranslateModelMigration;
use Termorize\Migrations\UserSettingMigration;
use Termorize\Migrations\VocabularyItemMigration;
use Termorize\Migrations\TranslationTaskMigration;

$kernel = new Termorize\Services\Kernel();
$kernel->connectDatabase();

const MIGRATIONS_CLASSES = [
    TelegramMigration::class,
    TranslateModelMigration::class,
    VocabularyItemMigration::class,
    UserSettingMigration::class,
    PendingTaskMigration::class,
    TranslationTaskMigration::class
];

foreach (MIGRATIONS_CLASSES as $migrationClass) {
    $migration = new $migrationClass();
    if (!Manager::schema()->hasTable($migration->getTable())) {
        $migration->migrate();

        echo "$migrationClass executed!\n";
    }
}

echo "Migrations processed\n";
