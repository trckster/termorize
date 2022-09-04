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