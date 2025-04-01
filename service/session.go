package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
)

var chatSessions = sync.Map{}

func GenerateSessionId() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	// return fmt.Sprintf("%s%d", hex.EncodeToString(b), time.Now().UnixNano())
	return hex.EncodeToString(b)
}

func SetChatSession(sessionId string, session context.CancelFunc) {
	chatSessions.Store(sessionId, session)
}

func GetChatSession(sessionId string) context.CancelFunc {
	cancelFunc, ok := chatSessions.Load(sessionId)
	if ok {
		return cancelFunc.(context.CancelFunc)
	}

	return nil
}

func RemoveChatSession(sessionId string) {
	chatSessions.Delete(sessionId)
}
