<?php

function stringToArray(string $str) : array
{
    $answer = [];

    for($i = 0;$i < strlen($str); ++$i)
    {
        $answer[] = $str[$i];
    }
    return $answer;
}

function getBasePath(string $subDir = ''): string
{
    return __DIR__."/../../".$subDir;
}