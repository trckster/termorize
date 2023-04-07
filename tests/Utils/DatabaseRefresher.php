<?php

namespace Tests\Utils;

use Illuminate\Database\Capsule\Manager;

class DatabaseRefresher
{
    public static function clearDatabase(): void
    {
        Manager::connection()->statement('SET foreign_key_checks=0');
        $databaseName = Manager::connection()->getDatabaseName();
        $tables = Manager::connection()->select("SELECT * FROM information_schema.tables WHERE table_schema = '$databaseName'");

        foreach ($tables as $table) {
            $name = $table->TABLE_NAME;
            Manager::table($name)->truncate();
        }

//        Manager::connection()->statement('SET foreign_key_checks=1');
        // TODO return back
    }
}
