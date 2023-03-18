<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;

class TranslateModelMigration
{
    public static function migrate(): void
    {
        $connection = Manager::connection();

        Manager::schema()->create('vocabulary_item', function ($table) {
            $table->integer('id')->unsigned();
            $table->increments('id');
            $table->primary('id');
            $table->ForeignId('translation_id');
            $table->integer('knowledge')->unsigned();
        });
    }
}
