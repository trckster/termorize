<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;

class VocabularyItemMigration
{
    public static function migrate(): void
    {
        Manager::schema()->create('vocabulary_items', function (Blueprint $table) {
            $table->id();

            $table->unsignedInteger('translation_id');
            $table->foreign('translation_id')
                ->references('id')
                ->on('translations');

            $table->unsignedInteger('knowledge');
        });
    }
}
