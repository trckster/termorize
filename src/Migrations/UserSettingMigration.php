<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;

class UserSettingMigration
{
    public static function migrate(): void
    {
        $connection = Manager::connection();

        Manager::schema()->create('user_setting', function ($table) {
            $table->ForeignId('user_id');
            $table->boolean('learns_vocabulary')->default(true);
        });
    }
}
