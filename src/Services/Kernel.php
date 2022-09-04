<?php

namespace Termorize\Services;

use Illuminate\Database\Capsule\Manager as Capsule;
use Longman\TelegramBot\Telegram;
use Longman\TelegramBot\Exception\TelegramException;

class Kernel
{
    public function run()
    {
        $botUsername = env('BOT_USERNAME');
        $botApiKey = env('BOT_API_KEY');

        $mysql_credentials = [
            'host'     => env("DATABASE_HOST"),
            'user'     => env("DATABASE_USERNAME"),
            'password' => env("DATABASE_PASSWORD"),
            'database' => env("DATABASE"),
        ];

        try {
            $telegram = new Telegram($botApiKey, $botUsername);

            $telegram->enableMySql($mysql_credentials);

            $response = $telegram->handleGetUpdates();
            $result = $response->getResult();
            $handler = new MessageHandler;

            foreach ($result as $update) {
                $handler->handle($update);
            }
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }
    public function connectDatabase(){
        $capsule = new Capsule;

        $capsule->addConnection([
            'driver' => 'mysql',
            'host' => env("DATABASE_HOST"),
            'database' => env("DATABASE"),
            'username' => env("DATABASE_USERNAME"),
            'password' => env("DATABASE_PASSWORD"),
            'charset' => 'utf8',
            'collation' => 'utf8_unicode_ci',
            'prefix' => '',
        ]);

        $capsule->setAsGlobal();
    }
}