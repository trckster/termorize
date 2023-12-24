<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;

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

            $table->boolean('learns_vocabulary')->default(true);
        });
    }
}
