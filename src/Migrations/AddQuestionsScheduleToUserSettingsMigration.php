<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;

class AddQuestionsScheduleToUserSettingsMigration implements MigrationInterface
{
    public function alreadyExecuted(): bool
    {
        return Manager::schema()->hasColumn('users_settings', 'questions_schedule');
    }

    public function migrate(): void
    {
        Manager::schema()->table('users_settings', function (Blueprint $table) {
            $table->jsonb('questions_schedule')->nullable();
        });
    }
}
