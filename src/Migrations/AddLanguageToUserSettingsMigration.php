<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;
use Termorize\Enums\Language;

class AddLanguageToUserSettingsMigration implements MigrationInterface
{
    public function alreadyExecuted(): bool
    {
        return Manager::schema()->hasColumn('users_settings', 'language');
    }

    public function migrate(): void
    {
        Manager::schema()->table('users_settings', function (Blueprint $table) {
            $table->string('language')->default(Language::en->name);
        });
    }
}
