<?php

namespace Tests\Unit\Commands;

use Termorize\Services\Kernel;
use Termorize\Services\Translator;
use Termorize\Models\Translation;
use Tests\TestCase;

class GetTranslationFromDBTest extends TestCase
{
    /**
     * @test
     */
    public function test()
    {
        $db = new Kernel();
        $originalText = "Hello";
        $db->connectDatabase();

        $translator = new Translator();
        $translationText = $translator->translate($originalText);


        $this->assertEquals(1, Translation::query()->where('original_text', $originalText)->count());

    }
}