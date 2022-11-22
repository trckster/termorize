<?php

namespace Tests\Unit\Commands;

use Tests\TestCase;
use Longman\TelegramBot\Request;
use Termorize\Commands\StartCommand;

class StartCommandTest extends TestCase
{
    /**
     * @test
     */
    public function example()
    {
        $mock = $this->makeAlias(Request::class);

        $mock->shouldReceive('sendMessage')
            ->once()
            ->with([
                'chat_id' => 5,
                'text' => 'Отправь мне любое слово и я его переведу.'
            ])->andReturn();

        $command = new StartCommand();

        $command->execute(5);

        $this->addToAssertionCount(1);
    }
}