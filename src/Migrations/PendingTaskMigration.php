<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;

class PendingTaskMigration
{
    public static function migrate(): void
    {
        $connection = Manager::connection();

        Manager::schema()->create('pending_task', function ($table) {
            $table->increments('id');
            $table->timestamp('scheduled_for');
            $table->timestamp('executed_at')->nullable();
            $table->string('method');
            $table->json('parameters');
            $table->enum('status', ['pending', 'success', 'failed']);
            $table->primary('id');
        });
    }
}
