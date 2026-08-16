package auth

import (
	"strings"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (s *Service) Login(c *gin.Context) {
	var req proto.LoginReq
	var rsp proto.Response

	// authentication disabled
	conf := config.GetInstance()
	if conf.Authentication == "disable" {
		rsp.OkRspWithData(c, &proto.LoginRsp{
			Token: "disabled",
		})
		return
	}

	clientIP := GetClientIP(c)
	if locked, code, msg := CheckLoginAttempt(clientIP); locked {
		time.Sleep(3 * time.Second)
		AuditFailure(c, "login", "locked out")
		rsp.ErrRsp(c, code, msg)
		return
	}

	if err := proto.ParseFormRequest(c, &req); err != nil {
		time.Sleep(3 * time.Second)
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	if ok := CompareAccount(req.Username, req.Password); !ok {
		time.Sleep(2 * time.Second)

		AuditFailure(c, "login", "invalid credentials")

		if locked, code, msg := RecordLoginFailure(clientIP); locked {
			rsp.ErrRsp(c, code, msg)
			return
		}

		rsp.ErrRsp(c, -2, "invalid username or password")
		return
	}

	// Second factor, only when the account has opted in.
	if TotpEnabled() {
		// Distinct code so the client knows to prompt, rather than reporting
		// the password as wrong. Reached only after the password verified.
		if strings.TrimSpace(req.Code) == "" {
			rsp.ErrRsp(c, -6, "verification code required")
			return
		}

		// A wrong code counts as a failed attempt, so the existing per-IP
		// lockout covers code guessing too.
		if !VerifyTotp(req.Code) {
			time.Sleep(2 * time.Second)

			AuditFailure(c, "login", "invalid totp code")

			if locked, code, msg := RecordLoginFailure(clientIP); locked {
				rsp.ErrRsp(c, code, msg)
				return
			}

			rsp.ErrRsp(c, -7, "invalid verification code")
			return
		}
	}

	ClearLoginAttempt(clientIP)
	Audit(c, "login", log.Fields{"username": req.Username})

	token, err := middleware.GenerateJWT(req.Username)
	if err != nil {
		time.Sleep(1 * time.Second)
		rsp.ErrRsp(c, -3, "generate token failed")
		return
	}

	rsp.OkRspWithData(c, &proto.LoginRsp{
		Token: token,
	})

	log.Debugf("login success, username: %s", req.Username)
}

func (s *Service) Logout(c *gin.Context) {
	conf := config.GetInstance()

	if conf.JWT.RevokeTokensOnLogout {
		config.RegenerateSecretKey()
	}

	var rsp proto.Response
	rsp.OkRsp(c)
}

func (s *Service) GetAccount(c *gin.Context) {
	var rsp proto.Response

	account, err := GetAccount()
	if err != nil {
		rsp.ErrRsp(c, -1, "get account failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetAccountRsp{
		Username: account.Username,
	})
	log.Debugf("get account successful")
}
