<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;

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
}
