<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;

class CloseQuestions implements CronCommand
{
    public function handle()
    {
        $pendingTasks = PendingTask::query()
            ->where('status', '=', 'Pending')
            ->get();

        foreach($pendingTasks as $pendingTask) {
            $pendingTaskTime = new Carbon($pendingTask->scheduled_for);
            if ($pendingTaskTime->diffInMinutes(Carbon::now()) <= 15) {
                $method = $pendingTask['method'];
                call_user_func($method, $pendingTask);
            }
        }
    }
}
