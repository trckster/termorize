<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;

class QuestionMigration implements MigrationInterface
{
    public function getTable(): string
    {
        return 'questions';
    }

    public function migrate(): void
    {
        Manager::schema()->create($this->getTable(), function (Blueprint $table) {
            $table->id();

            $table->unsignedBigInteger('vocabulary_item_id');
            $table->foreign('vocabulary_item_id')
                ->references('id')
                ->on('vocabulary_items');

            $table->bigInteger('chat_id');
            $table->unsignedBigInteger('message_id');

            $table->boolean('is_original');
            $table->boolean('is_answered')->default(false);

            $table->timestamps();
        });
    }
}