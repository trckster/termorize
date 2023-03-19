<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;

class VocabularyItem extends Model
{
    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'translation_id',
        'user_id',
        'knowledge',
    ];
}
