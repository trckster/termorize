<?php

function env(string $key) : string
{
return $_ENV[$key];
}

function stringToArray(string $str) : array
{
    $answer = [];

    for($i = 0;$i < strlen($str); ++$i)
    {
        $answer[] = $str[$i];
    }
    return $answer;
}