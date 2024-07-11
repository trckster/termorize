<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;

class AddQuestionsCountToUserSettingsMigration implements MigrationInterface
{
    public function alreadyExecuted(): bool
    {
        return Manager::schema()->hasColumn('users_settings', 'questions_count');
    }

    public function migrate(): void
    {
        Manager::schema()->table('users_settings', function (Blueprint $table) {
            $table->unsignedSmallInteger('questions_count')->default(1);
        });
    }
}
