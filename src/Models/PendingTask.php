<?php

namespace Termorize\Models;

use Carbon\Carbon;
use Illuminate\Database\Eloquent\Model;
use Termorize\Enums\PendingTaskStatus;
use Termorize\Services\Logger;
use Throwable;

/**
 * @property PendingTaskStatus::class $status
 * @property string $method
 * @property array $parameters
 * @property Carbon $scheduled_for
 * @property Carbon $executed_at
 */
class PendingTask extends Model
{
    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'status',
        'scheduled_for',
        'executed_at',
        'method',
        'parameters',
    ];

    protected $casts = [
        'status' => PendingTaskStatus::class,
        'parameters' => 'array',
        'scheduled_for' => 'datetime',
        'executed_at' => 'datetime',
    ];

    public function execute(): void
    {
        try {
            call_user_func($this->method, $this->parameters);
            $this->update([
                'status' => PendingTaskStatus::Success,
                'executed_at' => Carbon::now(),
            ]);
        } catch (Throwable $error) {
            Logger::info("There was an error while running {$this->method}: " . $error->getMessage());

            $this->update([
                'status' => PendingTaskStatus::Failed,
                'executed_at' => Carbon::now(),
            ]);
        }
    }
}
