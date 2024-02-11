<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int $chat_id
 * @property int $user_id
 */
class UserChat extends Model
{
    protected $table = 'user_chat';
    public const CREATED_AT = null;
    public const UPDATED_AT = null;
    protected $fillable = [
        'chat_id',
        'user_id',
    ];

}
