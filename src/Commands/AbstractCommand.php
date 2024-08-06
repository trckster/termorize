<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Entities\ServerResponse;
use Longman\TelegramBot\Entities\Update;
use Longman\TelegramBot\Request;
use Termorize\Models\User;

abstract class AbstractCommand
{
    const int MAX_MESSAGE_LENGTH = 4096;
    const int MAX_MESSAGES_AT_ONCE = 10;

    protected Update $update;

    abstract public function process(): void;

    public function setUpdate(Update $update): void
    {
        $this->update = $update;
    }

    protected function getClearedMessage(): string
    {
        $message = $this->update->getMessage()->getText();

        $messageParts = explode(' ', $message);
        array_shift($messageParts);

        return trim(implode(' ', $messageParts));
    }

    protected function getSenderId(): int
    {
        return $this->update->getMessage()->getFrom()->getId();
    }

    protected function loadUser(): User
    {
        return User::query()->find($this->getSenderId());
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
