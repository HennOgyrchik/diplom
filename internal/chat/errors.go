package chat

import "errors"

var AttemptsExceeded = errors.New("Превышено число попыток ввода")
var Close = errors.New("close")
var Timeout = errors.New("Превышено время ожидания ответа")
