package main

import (
	"errors"
	"strings"
)

type CallbackAction string

const (
	CBRemove            CallbackAction = "remove"
	CBCancel            CallbackAction = "cancel"
	CBCurrentToWishlist CallbackAction = "cur2wish"
	CBCurrentToHistory  CallbackAction = "cur2his"
)

func GetCallbackParamStr(action CallbackAction, data string) string {
	return string(action) + ":" + data
}

func GetCallbackParam(callbackData string) (CallbackAction, string, error) {
	cb := strings.Split(callbackData, ":")

	if len(cb) != 2 {
		return "", "", errors.New("Неизвестная операция: " + callbackData)
	}

	ca := CallbackAction(cb[0])
	switch ca {
	case CBRemove, CBCancel, CBCurrentToWishlist, CBCurrentToHistory:
		return ca, cb[1], nil
	}

	return "", "", errors.New("Неизвестное действие: " + callbackData)
}
