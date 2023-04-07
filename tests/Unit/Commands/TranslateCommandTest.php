<?php

namespace Tests\Unit\Commands;

use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Request;
use Mockery;
use Termorize\Commands\TranslateCommand;
use Termorize\Models\Translation;
use Termorize\Services\Translator;
use Termorize\Services\VocabularyItemService;
use Tests\TestCase;

class TranslateCommandTest extends TestCase
{
    /**
     * @test
     */
    public function translateCommandSendsTranslation()
    {
        $update = $this->mockCascade([
            '__class' => Update::class,
            'getMessage' => [
                'getText' => 'Hello',
                'getChat' => [
                    'getId' => 5,
                ],
                'getFrom' => [
                    'getId' => 77,
                ],
            ],
        ]);

        $mockRequest = $this->makeAlias(Request::class);
        $mockRequest->shouldReceive('sendMessage')
            ->once()
            ->with([
                'chat_id' => 5,
                'text' => 'Здравствуйте',
            ])->andReturn();

        $translation = Translation::query()->create([
            'original_text' => 'Hello',
            'translation_text' => 'Здравствуйте',
            'original_lang' => 'en',
            'translation_lang' => 'ru',
        ]);

        $translatorMock = Mockery::mock(Translator::class);
        $translatorMock->shouldReceive('translate')
            ->withArgs(['Hello'])
            ->once()
            ->andReturn($translation);

        $vocabularyMock = Mockery::mock(VocabularyItemService::class);
        $vocabularyMock->shouldReceive('save')
            ->withArgs([$translation, 77])
            ->once();

        $command = new TranslateCommand();
        $this->mockPrivateProperty($command, 'translator', $translatorMock);
        $this->mockPrivateProperty($command, 'vocabularyService', $vocabularyMock);
        $command->setUpdate($update);
        $command->process();

        $this->addToAssertionCount(1);
    }
}
