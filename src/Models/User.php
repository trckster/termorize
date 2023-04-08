<?php

namespace Termorize\Models;

use Carbon\Carbon;
use Illuminate\Database\Eloquent\Model;

/**
 * @property int $id
 * @property bool $is_bot
 * @property string $first_name
 * @property string $last_name
 * @property string $username
 * @property string $language_code
 * @property bool $is_premium
 * @property bool $added_to_attachment_menu
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class User extends Model
{
    protected $fillable = [
        'is_bot',
        'first_name',
        'last_name',
        'username',
        'language_code',
        'is_premium',
        'added_to_attachment_menu',
        'created_at',
        'updated_at',
    ];
    protected $table = 'user';
}
