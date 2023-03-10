<?php

namespace Tests\Unit\Commands;

use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Request;
use Termorize\Commands\DefaultCommand;
use Tests\TestCase;

class DefaultCommandTest extends TestCase
{
    /**
     * @test
     */
    public function test()
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
                'text' => 'Такой команды нет, попробуйте ввести другую'
            ])->andReturn();

        $command = new DefaultCommand();
        $command->setUpdate($update);
        $command->process();

        $this->addToAssertionCount(1);
    }
}
