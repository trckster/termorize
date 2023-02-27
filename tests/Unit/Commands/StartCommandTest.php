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
    public function test()
    {
        $update = $this->mockCascade([
            'getMessage' => [
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
                'text' => 'Отправь мне любое слово и я его переведу.'
            ])->andReturn();

        $command = new StartCommand();
        $command->setUpdate($update);
        $command->process();

        $this->addToAssertionCount(1);
    }
}
