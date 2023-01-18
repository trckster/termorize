<?php

namespace Tests\Unit\Services;

use Termorize\Services\LanguageIdentifier;
use Tests\TestCase;

class LanguageIdentifierEnglishTest extends TestCase
{
    /**
     * @test
     */
    public function test()
    {

        $text = 'hello';
        $textLang = LanguageIdentifier::identify($text);

        $this->assertEquals('en', $textLang);

    }
}