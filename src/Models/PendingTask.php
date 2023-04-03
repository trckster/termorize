<?php

namespace Termorize\Models;

use Carbon\Carbon;
use Illuminate\Database\Eloquent\Model;
use Termorize\Enums\PendingTaskStatus;

/**
 * @property PendingTaskStatus::class $status
 * @property string $method
 * @property string $parameters
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
    ];
}
