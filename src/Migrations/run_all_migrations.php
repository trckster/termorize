<?php

require_once './vendor/autoload.php';

use Illuminate\Database\Capsule\Manager;
use Termorize\Migrations\PendingTaskMigration;
use Termorize\Migrations\TelegramMigration;
use Termorize\Migrations\TranslateModelMigration;
use Termorize\Migrations\UserSettingMigration;
use Termorize\Migrations\VocabularyItemMigration;

$kernel = new Termorize\Services\Kernel();
$kernel->connectDatabase();

TelegramMigration::migrate();

$migrationsClasses = [
    TranslateModelMigration::class,
    VocabularyItemMigration::class,
    UserSettingMigration::class,
    PendingTaskMigration::class,
];

foreach($migrationsClasses as $migrationClass){
    $migration = new $migrationClass();
    if (!Manager::schema()->hasTable($migration->getTable())){
        $migration->migrate();
    }
}

echo "Migrations processed\n";
