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

    protected $table = 'users_settings';

    public static function createDefaultSetting(User $user)
    {
        return self::query()
            ->create([
                'user_id' => $user->id,
                'learns_vocabulary' => true,
            ]);
    }
}
