<?php

namespace Termorize\Models;

use Illuminate\Database\Eloquent\Model;

/**
 * @property int $user_id
 * @property bool $learns_vocabulary
 */
class UserSetting extends Model
{
    protected $table = 'users_settings';
    protected $primaryKey = 'user_id';
    public $incrementing = false;

    public const CREATED_AT = null;
    public const UPDATED_AT = null;

    protected $fillable = [
        'user_id',
        'learns_vocabulary',
    ];

    public static function createDefaultSetting(User $user): self
    {
        return self::query()
            ->create([
                'user_id' => $user->id,
                'learns_vocabulary' => true,
            ]);
    }
}
