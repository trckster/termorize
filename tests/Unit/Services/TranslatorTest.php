<?php

namespace Tests\Unit\Commands;

use Termorize\Services\Kernel;
use Termorize\Services\Translator;
use Termorize\Models\Translation;
use Tests\TestCase;

class TranslatorTest extends TestCase
{
    /**
     * @test
     */
    public function test()
    {
        $db = new Kernel();
        $originalText = "Hello";
        $correctTranslate = 'Здравствуйте';
        $db->connectDatabase();

        $translator = new Translator();
        $translationText = $translator->translate($originalText);


        $this->assertEquals($correctTranslate, Translation::query()->where('original_text', $originalText)->value('translation_text'));

    }
}