<?php

namespace Termorize\Services;

use Carbon\Carbon;

class Logger
{
    public static function info(string $message): void
    {
        $time = Carbon::now()->toDateTimeString();

        echo "[$time] $message\n";
    }
}