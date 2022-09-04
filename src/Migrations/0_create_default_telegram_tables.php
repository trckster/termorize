<?php

namespace Termorize\Migrations;
use Illuminate\Support\Facades\DB;
class TelegramMigration{
    public static function migrate(){
        $rawQuery = file_get_contents("../../vendor/longman/telegram-bot/structure.sql");
        DB::raw($rawQuery);
    }
}

