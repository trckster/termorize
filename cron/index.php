<?php

require_once __DIR__ . '/../vendor/autoload.php';

use Termorize\Services\Kernel;

if ($argc < 2) {
    throw new Error('Not enough arguments');
}

$className = $argv[1];

$kernel = new Kernel;
$kernel->connectDatabase();

$classesInCron = scandir(getBasePath('src/Cron'));
foreach ($classesInCron as $fileName) {
    if ($fileName === "$className.php") {
        $className = "Termorize\Cron\\$className";
        $command = new $className;
        $command->handle();
    }
}
