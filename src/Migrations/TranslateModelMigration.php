<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;

class TranslateModelMigration
{

    public static function migrate()
    {
        $connection = Manager::connection();

        Manager::schema()->create('translations', function ($table){
            $table->increments('id');
            $table->string('original_text');
            $table->string('translation_text');
            $table->string('original_lang');
            $table->string('translation_lang');
        });
    }

}