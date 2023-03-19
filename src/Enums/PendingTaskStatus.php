<?php

namespace Termorize\Enums;

enum PendingTaskStatus: string
{
    case Pending = 'Pending';
    case Success = 'Success';
    case Failed = 'Failed';
}