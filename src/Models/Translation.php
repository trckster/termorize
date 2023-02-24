<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;

class Translation extends Model
{
    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'original_text',
        'translation_text',
        'original_lang',
        'translation_lang',
    ];
}
