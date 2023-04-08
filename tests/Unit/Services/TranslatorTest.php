<?php

namespace Tests\Unit\Services;

use GuzzleHttp\Client;
use Mockery;
use Psr\Http\Message\ResponseInterface;
use Termorize\Models\Translation;
use Termorize\Services\Translator;
use Tests\TestCase;

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
        $contents = json_encode(['text' => [$correctTranslation]]);
        $mock = $this->mockCascade([
            '__class' => Client::class,
            'get' => [
                '__class' => ResponseInterface::class,
                'getBody' => [
                    'getContents' => $contents,
                ],
            ],
        ]);
        $this->mockPrivateProperty($translator, 'httpClient', $mock);

        $result = $translator->translate($originalWord)->translation_text;

        $this->assertEquals($result, $correctTranslation);

        $translation = Translation::query()->first();

        $this->assertNotNull($translation);

        $this->assertEquals($correctTranslation, $translation->translation_text);
        $this->assertEquals($originalWord, $translation->original_text);
        $this->assertEquals('en', $translation->translation_lang);
        $this->assertEquals('ru', $translation->original_lang);
    }

    /**
     * @test
     */
    public function canUseCacheWhenRequestingTheSameTranslation()
    {
        $translator = new Translator();

        $originalWord = 'привет';
        $correctTranslation = 'hello';
        $contents = json_encode(['text' => [$correctTranslation]]);
        $mock = $this->mockCascade([
            '__class' => Client::class,
            'get' => [
                '__class' => ResponseInterface::class,
                'getBody' => [
                    'getContents' => $contents,
                ],
            ],
        ]);

        $this->mockPrivateProperty($translator, 'httpClient', $mock);

        $result = $translator->translate($originalWord)->translation_text;

        $mock = Mockery::mock(Client::class);
        $mock->shouldNotReceive('get');

        $this->mockPrivateProperty($translator, 'httpClient', $mock);

        $secondResult = $translator->translate($originalWord)->translation_text;

        $this->assertEquals($result, $secondResult);
        $this->assertEquals($result, $correctTranslation);
    }
}
