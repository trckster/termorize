<?php

namespace Tests\Unit\Commands;

use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Request;
use Termorize\Commands\TranslateCommand;
use Tests\TestCase;

class TranslateCommandTest extends TestCase
{
    /**
    *@test
    */
    public function TranslateCommandWorks()
    {
        $update = $this->mockCascade([
            '__class' => Update::class,
            'getMessage' => [
                'getText' => 'Hello',
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
                'text' => 'Здравствуйте'
            ])->andReturn();

        $command = new TranslateCommand();
        $command->setUpdate($update);
        $command->process();

        $this->addToAssertionCount(1);
    }
}
