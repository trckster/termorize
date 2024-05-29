<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;

class AddIsCustomToTranslationsMigration implements MigrationInterface
{
    public function alreadyExecuted(): bool
    {
        return Manager::schema()->hasColumn('translations', 'is_custom');
    }

    public function migrate(): void
    {
        Manager::schema()->table('translations', function (Blueprint $table) {
            $table->boolean('is_custom')->default(false);
        });
    }
}