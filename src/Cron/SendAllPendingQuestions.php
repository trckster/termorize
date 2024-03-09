<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;

class SendAllPendingQuestions implements CronCommand
{
    public function handle()
    {
        echo "Works\n";

        $pendingTasks = PendingTask::query()
            ->where('status', PendingTaskStatus::Pending)
            ->get();

        foreach($pendingTasks as $pendingTask) {
            $pendingTaskTime = $pendingTask->scheduled_for;
            if ($pendingTaskTime->diffInMinutes(Carbon::now()) <= 15) {
                $method = $pendingTask['method'];
                call_user_func($method, $pendingTask);
            }
        }
    }
}
