<?php

namespace Termorize\Services;

require_once getBasePath('vendor/autoload.php');

use Dotenv\Dotenv;
use Illuminate\Database\Capsule\Manager as Capsule;
use Longman\TelegramBot\Exception\TelegramException;
use Longman\TelegramBot\Telegram;

class Kernel
{
    public function run(): void
    {
        $botUsername = env('BOT_USERNAME');
        $botApiKey = env('BOT_API_KEY');

        $mysql_credentials = [
            'host'     => env('DATABASE_HOST'),
            'user'     => env('DATABASE_USERNAME'),
            'password' => env('DATABASE_PASSWORD'),
            'database' => env('DATABASE'),
        ];

        $this->connectDatabase();

        try {
            $telegram = new Telegram($botApiKey, $botUsername);

            $telegram->enableMySql($mysql_credentials);

            $response = $telegram->handleGetUpdates();
            $result = $response->getResult();
            $handler = new MessageHandler();

            foreach ($result as $update) {
                $handler->handle($update);
            }
        } catch (TelegramException $e) {
            echo $e->getMessage();
        }
    }

    public function connectDatabase(): void
    {
        if (empty($_ENV)) {
            $dotenv = Dotenv::createImmutable(__DIR__ . '/../../');
            $dotenv->load();
        }

        $capsule = new Capsule();

        $capsule->addConnection([
            'driver' => 'mysql',
            'host' => env('DATABASE_HOST'),
            'database' => env('DATABASE'),
            'username' => env('DATABASE_USERNAME'),
            'password' => env('DATABASE_PASSWORD'),
            'charset' => 'utf8',
            'collation' => 'utf8_unicode_ci',
            'prefix' => '',
        ]);

        $capsule->setAsGlobal();
        $capsule->bootEloquent();
    }
}
