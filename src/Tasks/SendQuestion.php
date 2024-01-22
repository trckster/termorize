<?php

namespace Termorize\Tasks;

use Longman\TelegramBot\Request;
use Termorize\Models\PendingTask;
use Termorize\Models\User;

class SendQuestion
{
    public function handle(PendingTask $pendingTask)
    {
        $params = json_decode($pendingTask->parameters, true);
        /*$user = User::query()->where('id', '=', $params[]);

        Request::sendMessage([
            'chat_id' =>
        ])*/
    }
}
