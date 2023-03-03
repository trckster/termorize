<?php

namespace Tests\Utils;

use Dotenv\Dotenv;
use Illuminate\Database\Capsule\Manager;
use Illuminate\Support\Facades\DB;


class DatabaseRefresher
{
    public static function clearDatabase()
    {
        Manager::connection()->statement("SET foreign_key_checks=0");
        $databaseName = Manager::connection()->getDatabaseName();
        $tables = Manager::connection()->select("SELECT * FROM information_schema.tables WHERE table_schema = '$databaseName'");
        foreach ($tables as $table) {
            $name = $table->TABLE_NAME;
            //if you don't want to truncate migrations
            if ($name == 'migrations') {
                continue;
            }
            Manager::table($name)->truncate();
        }
        Manager::connection()->statement("SET foreign_key_checks=1");
    }
}