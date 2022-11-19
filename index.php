<?php

require_once getBasePath('vendor/autoload.php');


$dotenv = Dotenv\Dotenv::createImmutable(__DIR__);
$dotenv->load();

$kernel = new Termorize\Services\Kernel();

$kernel->run();
