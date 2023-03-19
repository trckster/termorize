<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;
use Illuminate\Database\Schema\Blueprint;

class UserSettingMigration
{
    public static function migrate(): void
    {
        Manager::schema()->create('users_settings', function (Blueprint $table) {
            $table->bigInteger('user_id');
            $table->foreign('user_id')
                ->references('id')
                ->on('user');

            $table->boolean('learns_vocabulary')->default(true);
        });
    }
}
