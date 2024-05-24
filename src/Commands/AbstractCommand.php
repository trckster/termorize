<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Entities\ServerResponse;
use Longman\TelegramBot\Entities\Update;
use Termorize\Models\User;
use Longman\TelegramBot\Request;

abstract class AbstractCommand
{
    protected Update $update;

    abstract public function process(): void;

    public function setUpdate(Update $update): void
    {
        $this->update = $update;
    }

    protected function loadUser(): User
    {
        return User::query()->find($this->update->getMessage()->getFrom()->getId());
    }

    protected function reply(string $text, array $options = []): ServerResponse
    {
        return Request::sendMessage([
            'chat_id' => $this->update->getMessage()->getChat()->getId(),
            'text' => $text,
            ...$options,
        ]);
    }
}
