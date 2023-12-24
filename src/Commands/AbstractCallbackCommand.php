<?php

namespace Termorize\Commands;

abstract class AbstractCallbackCommand extends AbstractCommand
{
    protected array $callbackData;

    public function parseCallbackData(): void
    {
        $this->callbackData = json_decode($this->update->getCallbackQuery()->getData(), true);
    }
}
