<?php

namespace Termorize\Models;

use Carbon\Carbon;
use Illuminate\Database\Eloquent\Collection;
use Illuminate\Database\Eloquent\Model;

/** @property int $id
 *  @property  string $text
 *  @property int $chatId
 *  */

class Message extends Model {
    protected $table = 'message';

    protected $fillable = [];
}