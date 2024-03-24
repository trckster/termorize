<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

/**
 * @property int $id
 * @property int $message_id
 * @property int $vocabulary_item_id
 */
class TranslationTasks extends Model
{
    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'message_id',
        'vocabulary_item_id'
    ];
    
}
