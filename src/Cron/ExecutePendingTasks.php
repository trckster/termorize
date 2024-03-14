<?php

namespace Termorize\Cron;

use Carbon\Carbon;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Interfaces\CronCommand;
use Termorize\Models\PendingTask;

class ExecutePendingTasks implements CronCommand
{
    public function handle(): void
    {
        PendingTask::query()
            ->where('status', PendingTaskStatus::Pending)
            ->where('scheduled_for', '<=', Carbon::now())
            ->get()
            ->each
            ->execute();
    }
}
