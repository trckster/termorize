<?php

namespace Tests\Unit\Services;

use Termorize\Services\LanguageIdentifier;
use Tests\TestCase;

class LanguageIdentifierRussianTest extends TestCase
{
    /**
     * @test
     */
    public function test()
    {

        $text = 'привет';
        $textLang = LanguageIdentifier::identify($text);

        $this->assertEquals('ru', $textLang);

    }
}