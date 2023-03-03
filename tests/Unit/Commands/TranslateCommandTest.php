<?php

namespace Tests\Unit\Commands;

use Longman\TelegramBot\Entities\Update;
use Termorize\Services\Kernel;
use Longman\TelegramBot\Request;
use Termorize\Commands\TranslateCommand;
use Tests\TestCase;

class TranslateCommandTest extends TestCase
{
    public function test()
    {
        $update = $this->mockCascade([
            'getMessage' => [
                'getText' => "Hello",
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
                'text' => "Здравствуйте"
            ])->andReturn();

        $command = new TranslateCommand();
        $command->setUpdate($update);
        $command->process();

        $this->addToAssertionCount(1);
    }
}
