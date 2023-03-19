<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;
use Termorize\Enums\PendingTaskStatus;

class PendingTaskMigration
{
    public static function migrate(): void
    {
        Manager::schema()->create('pending_tasks', function (Blueprint $table) {
            $table->id();
            $table->enum('status', [
                PendingTaskStatus::Pending->value,
                PendingTaskStatus::Success->value,
                PendingTaskStatus::Failed->value,
            ]);
            $table->timestamp('scheduled_for');
            $table->timestamp('executed_at')->nullable();
            $table->string('method');
            $table->jsonb('parameters');
        });
    }
}
