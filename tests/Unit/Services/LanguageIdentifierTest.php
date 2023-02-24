<?php

namespace Tests\Unit\Services;

use Termorize\Services\LanguageIdentifier;
use Tests\TestCase;

class LanguageIdentifierTest extends TestCase
{
    /**
     * @test
     */
    public function canDetectRussian()
    {
        $text = 'привет';
        $textLang = LanguageIdentifier::identify($text);

        $this->assertEquals('ru', $textLang);
    }

    /**
     * @test
     */
    public function canDetectEnglish()
    {
        $text = 'hello';
        $textLang = LanguageIdentifier::identify($text);

        $this->assertEquals('en', $textLang);
    }
}