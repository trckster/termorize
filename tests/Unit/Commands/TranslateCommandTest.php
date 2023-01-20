<?php

namespace Tests\Unit\Commands;

use Longman\TelegramBot\Entities\Update;
use Termorize\Services\Kernel;
use Longman\TelegramBot\Request;
use Termorize\Commands\TranslateCommand;
use Termorize\Models\Translation;
use Tests\TestCase;

class TranslateCommandTest extends TestCase
{
    /**
     * @test
     */
    public function test()
    {
        $db = new Kernel();
        $originalText = "Hello";
        $translationText = "Здравствуйте";
        $db->connectDatabase();
        $update = $this->mockCascade([
            'getMessage' => [
                'getText' => $originalText,
                'getChat' => [
                    'getId' => 5,
                ],
            ],
        ], Update::class);

        $mock = $this->makeAlias(Request::class);
        $mock->shouldReceive('sendMessage')
            ->once()
            ->with([
                'chat_id' => 5,
                'text' => $translationText
            ])->andReturn();

        $command = new TranslateCommand();
        $command->setUpdate($update);
        $command->process();

        $this->addToAssertionCount(1);

    }
}