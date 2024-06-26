<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;

class TelegramMigration implements MigrationInterface
{
    public function alreadyExecuted(): bool
    {
        return false;
    }

    public function migrate(): void
    {
        $rawQuery = file_get_contents(getBasePath('vendor/longman/telegram-bot/structure.sql'));
        $connection = Manager::connection();

        foreach (explode(";\n", $rawQuery) as $query) {
            if (!empty($query)) {
                $connection->statement($query);
            }
        }
    }
}
