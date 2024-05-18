<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Entities\Update;
use Termorize\Models\User;

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
}
