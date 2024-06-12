package rpc

import (
	"encoding/json"
	"gateway/log"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const SessionTimeout = 60 * 10

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET,PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		// c.Next()
	}
}

func UserSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)
		sessionId, err := c.Cookie(SessionCookieName)
		defer func() {
			log.Debug("session id in cookie", sessionId)
			c.SetCookie(SessionCookieName, sessionId, 3600, "/", Host, false, false)
			c.Set(SesssionIdContextName, sessionId)
			status, _ := json.Marshal(&UserStatus{"a", "a", "", 0, ""})
			sess.Set(sessionId, string(status))
			sess.Save()
			// c.Next()
		}()
		if err != nil {
			//new session id
			log.Error("get session id from cookie error", err)
			sessionId = uuid.New().String()
			return
		}
		stat := sess.Get(sessionId)
		if stat == nil || stat == emptyStatus {
			log.Info("no session yet")
			return
		}
		statusStr, ok := stat.(string)
		log.Debug("session status", statusStr)
		var status UserStatus
		if !ok {
			log.Error("user status in sessionId struct incorrect", sessionId)
			return
		}
		err = json.Unmarshal([]byte(statusStr), &status)
		if err != nil {
			log.Error("unmarshal user status incorrect", statusStr)
			return
		}
		if (time.Now().Unix() - status.LastTime) > SessionTimeout {
			return
		}

		c.Set(LastMessageContextName, status.MessageId)
		c.Set(LastConversationContextName, status.ConversationId)
		c.Set(LastRelayUrlContextName, status.Url)
		c.Set(LastModelName, status.Model)
	}
}
