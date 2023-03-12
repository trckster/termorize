<?php

namespace Tests\Unit\Commands;

use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Request;
use Termorize\Commands\StartCommand;
use Tests\TestCase;

class StartCommandTest extends TestCase
{
    /**
     * @test
     */
    public function startCommandWorks()
    {
        $update = $this->mockCascade([
            '__class' => Update::class,
            'getMessage' => [
                'getChat' => [
                    'getId' => 5,
                ],
            ],
        ]);

        $mock = $this->makeAlias(Request::class);
        $mock->shouldReceive('sendMessage')
            ->once()
            ->with([
                'chat_id' => 5,
                'text' => 'Отправь мне любое слово и я его переведу.',
            ])->andReturn();

        $command = new StartCommand();
        $command->setUpdate($update);
        $command->process();

        $this->addToAssertionCount(1);
    }
}
