<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;

class TranslationTaskMigration implements MigrationInterface
{
    public function getTable(): string
    {
        return 'translation_tasks';
    }

    public function migrate(): void
    {
        Manager::schema()->create($this->getTable(), function ($table) {
            $table->increments('id');

            $table->bigInteger('vocabulary_item_id');
            $table->foreign('vocabulary_item_id')
                ->references('id')
                ->on('vocabulary_items');

            $table->bigInteger('user_id');
            $table->foreign('user_id')
                ->references('id')
                ->on('user');
        });

    }
}
