<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;
use Termorize\Enums\UserStatus;

class UserSettingMigration implements MigrationInterface
{
    public function getTable(): string
    {
        return 'users_settings';
    }

    public function migrate(): void
    {
        Manager::schema()->create($this->getTable(), function (Blueprint $table) {
            $table->bigInteger('user_id');
            $table->foreign('user_id')
                ->references('id')
                ->on('user');
            $table->enum('status', [
                UserStatus::AddingWords->value,
                UserStatus::Answering->value,
            ]
            )->default(UserStatus::AddingWords);

            $table->boolean('learns_vocabulary')->default(true);
        });
    }
}
