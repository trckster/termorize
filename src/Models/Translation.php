<?php

use Illuminate\Database\Eloquent\Model;

class Translation extends Model
{

    protected $fillable = [
        'original_text', 'translation_text','original_lang','translation_lang'
    ];

}