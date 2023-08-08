<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Entities\CallbackQuery;

abstract class AbstractCallbackCommand
{
    protected CallbackQuery $callbackQuery;

    abstract public function process(): void;

    public function setCallbackQuery(CallbackQuery $callbackQuery): void
    {
        $this->callbackQuery = $callbackQuery;
    }
}
