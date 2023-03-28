<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int $user_id
 * @property bool $learns_vocabulary
 */
class UserSetting extends Model
{
    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'user_id',
        'learns_vocabulary',
    ];
}
