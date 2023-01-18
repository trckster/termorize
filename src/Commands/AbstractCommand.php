<?php

namespace Termorize\Commands;

use Longman\TelegramBot\Entities\Update;

abstract class AbstractCommand
{
    protected Update $update;

    abstract public function process(): void;

    public function setUpdate(Update $update): void
    {
        $this->update = $update;
    }
}
