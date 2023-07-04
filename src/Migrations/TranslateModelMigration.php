<?php

namespace Termorize\Migrations;

use Illuminate\Database\Capsule\Manager;

class TranslateModelMigration
{
    public function getTable(): string
    {
        return 'translations';
    }

    public function migrate(): void
    {
        Manager::schema()->create($this->getTable(), function ($table) {
            $table->increments('id');
            $table->text('original_text');
            $table->text('translation_text');
            $table->string('original_lang');
            $table->string('translation_lang');
        });
    }
}
