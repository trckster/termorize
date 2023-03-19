<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;
use Termorize\Enums\PendingTaskStatus;

class PendingTask extends Model
{
    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'status',
        'scheduled_for',
        'executed_at',
        'method',
        'parameters'
    ];

    protected $casts = [
        'status' => PendingTaskStatus::class,
    ];
}