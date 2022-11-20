<?php

namespace Unit\Commands;

use PHPUnit\Framework\TestCase;
use Termorize\Commands\StartCommand;

class StartCommandTest extends TestCase
{
    /**
     * @test
     */
    public function example()
    {
        $mock = \Mockery::mock('alias:Longman\TelegramBot\Request');

        $mock->shouldReceive('sendMessage')
            ->withArgs([
                [
                    'chat_id' => 5,
                    'text' => 'Отправь мне любое слово и я его переведу.'
                ]
            ])->andReturnUndefined();

        $command = new StartCommand();

        $this->assertNull($command->execute(5));
    }
}