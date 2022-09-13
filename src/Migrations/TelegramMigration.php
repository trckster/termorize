<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;


class TelegramMigration
{
    public static function migrate()
    {
        $rawQuery = file_get_contents("../../vendor/longman/telegram-bot/structure.sql");
        $connection = Manager::connection();

        foreach (explode(";\n", $rawQuery) as $query) {
            if (!empty($query)) {
                $connection->statement($query);
            }
        }
    }
}

