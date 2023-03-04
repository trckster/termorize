<?php

namespace Tests\Unit\Services;

use GuzzleHttp\Client;
use Termorize\Services\Kernel;
use Termorize\Services\Translator;
use Termorize\Models\Translation;
use Tests\TestCase;
use Mockery;

class TranslatorTest extends TestCase

{
    /**
     * @test
     */
    public function canSaveTranslationInDatabase()
    {
        $translator = new Translator();

        $originalWord = 'привет';
        $correctTranslation = 'hello';
        $contents = json_encode(['text' => [$correctTranslation]]);;
        $mock = $this->mockCascade([
            'get' => [
                'getBody' => [
                    'getContents' => $contents
                ]
            ]
        ], Client::class);
        $this->mockPrivateProperty($translator, 'httpClient', $mock);
        
        $result = $translator->translate($originalWord);

        $this->assertEquals($result, $correctTranslation);

        $translation = Translation::query()->first();

        $this->assertNotNull($translation);

        // Assert that database has this translation
    }

    public function canUseCacheWhenRequestingTheSameTranslation()
    {
        // Ask translation service to translate a word

        // Ask translation service to translate the same word

        // Assert that database was used and no requests were sent
    }
}
