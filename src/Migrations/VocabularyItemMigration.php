<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;

class VocabularyItemMigration
{
    public function getTable(): string
    {
        return 'vocabulary_items';
    }

    public function migrate(): void
    {
        Manager::schema()->create($this->getTable(), function (Blueprint $table) {
            $table->id();
            $table->bigInteger('user_id');
            $table->foreign('user_id')
                ->references('id')
                ->on('user');
            $table->unsignedInteger('translation_id');
            $table->foreign('translation_id')
                ->references('id')
                ->on('translations');

            $table->unsignedInteger('knowledge');
        });
    }
}
