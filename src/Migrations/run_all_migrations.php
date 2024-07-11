<?php

require_once './vendor/autoload.php';

use Termorize\Migrations\AddIsCustomToTranslationsMigration;
use Termorize\Migrations\AddLanguageToUserSettingsMigration;
use Termorize\Migrations\AddQuestionsCountToUserSettingsMigration;
use Termorize\Migrations\AddQuestionsScheduleToUserSettingsMigration;
use Termorize\Migrations\MigrationInterface;
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
    AddQuestionsCountToUserSettingsMigration::class,
    AddIsCustomToTranslationsMigration::class,
    AddQuestionsScheduleToUserSettingsMigration::class,
    AddLanguageToUserSettingsMigration::class,
];

foreach (MIGRATIONS_CLASSES as $migrationClass) {
    /** @var MigrationInterface $migration */
    $migration = new $migrationClass();

    if (!$migration->alreadyExecuted()) {
        $migration->migrate();

        Logger::info("$migrationClass executed!");
    }
}

Logger::info('Migrations processed');
