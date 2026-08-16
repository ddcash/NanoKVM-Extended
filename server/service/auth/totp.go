package auth

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"NanoKVM-Server/proto"
)

// TOTP is opt-in and never required. A device with no totp.json behaves
// exactly as before.
//
// Recovery matters here: this device has no shell exposed by default, so
// losing the authenticator with no backup codes left would otherwise mean
// reflashing. To recover, power the device down, read the SD card on another
// machine and delete /etc/kvm/totp.json.
const (
	TotpConfigFile = "/etc/kvm/totp.json"
	totpIssuer     = "NanoKVM"
	backupCodeQty  = 10
)

var totpMu sync.Mutex

type TotpConfig struct {
	Enabled bool   `json:"enabled"`
	Secret  string `json:"secret"`
	// bcrypt hashes; a code is removed as it is consumed.
	BackupCodes []string `json:"backupCodes"`
}

// TotpEnabled reports whether a successful password check still needs a code.
func TotpEnabled() bool {
	cfg, err := loadTotpConfig()
	if err != nil {
		// Fail closed only if the file exists but cannot be read: treating an
		// unreadable config as "disabled" would let a corrupted file silently
		// remove the second factor.
		return errors.Is(err, errTotpUnreadable)
	}
	return cfg.Enabled && cfg.Secret != ""
}

var errTotpUnreadable = errors.New("totp config unreadable")

// VerifyTotp accepts either a current TOTP code or an unused backup code. A
// backup code is consumed on use.
func VerifyTotp(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}

	totpMu.Lock()
	defer totpMu.Unlock()

	cfg, err := loadTotpConfigLocked()
	if err != nil || !cfg.Enabled {
		return false
	}

	if totp.Validate(code, cfg.Secret) {
		return true
	}

	for i, hashed := range cfg.BackupCodes {
		if bcrypt.CompareHashAndPassword([]byte(hashed), []byte(code)) == nil {
			cfg.BackupCodes = append(cfg.BackupCodes[:i], cfg.BackupCodes[i+1:]...)
			if err := saveTotpConfigLocked(cfg); err != nil {
				log.Errorf("failed to consume backup code: %s", err)
			}
			log.Warnf("totp backup code used, %d remaining", len(cfg.BackupCodes))
			return true
		}
	}

	return false
}

func (s *Service) GetTotp(c *gin.Context) {
	var rsp proto.Response

	cfg, err := loadTotpConfig()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		rsp.ErrRsp(c, -1, "failed to read totp config")
		return
	}

	rsp.OkRspWithData(c, &proto.GetTotpRsp{
		Enabled: cfg.Enabled,
		Pending: !cfg.Enabled && cfg.Secret != "",
	})
}

// SetupTotp generates a secret and returns the enrolment URI. The second
// factor is not active until EnableTotp confirms a code, so a botched
// enrolment cannot lock anyone out.
func (s *Service) SetupTotp(c *gin.Context) {
	var rsp proto.Response

	if TotpEnabled() {
		rsp.ErrRsp(c, -1, "totp is already enabled")
		return
	}

	account, err := GetAccount()
	if err != nil {
		rsp.ErrRsp(c, -2, "failed to read account")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: account.Username,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		log.Errorf("failed to generate totp secret: %s", err)
		rsp.ErrRsp(c, -3, "failed to generate secret")
		return
	}

	totpMu.Lock()
	err = saveTotpConfigLocked(&TotpConfig{Enabled: false, Secret: key.Secret()})
	totpMu.Unlock()
	if err != nil {
		log.Errorf("failed to save totp config: %s", err)
		rsp.ErrRsp(c, -4, "failed to save secret")
		return
	}

	Audit(c, "totp_setup", nil)

	rsp.OkRspWithData(c, &proto.SetupTotpRsp{
		Uri:    key.URL(),
		Secret: key.Secret(),
	})
}

func (s *Service) EnableTotp(c *gin.Context) {
	var req proto.EnableTotpReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	totpMu.Lock()
	defer totpMu.Unlock()

	cfg, err := loadTotpConfigLocked()
	if err != nil || cfg.Secret == "" {
		rsp.ErrRsp(c, -2, "run setup first")
		return
	}
	if cfg.Enabled {
		rsp.ErrRsp(c, -3, "totp is already enabled")
		return
	}

	// Confirms the authenticator is actually in sync before the factor starts
	// being enforced.
	if !totp.Validate(strings.TrimSpace(req.Code), cfg.Secret) {
		AuditFailure(c, "totp_enable", "invalid code")
		rsp.ErrRsp(c, -4, "invalid code")
		return
	}

	plain, hashed, err := generateBackupCodes()
	if err != nil {
		log.Errorf("failed to generate backup codes: %s", err)
		rsp.ErrRsp(c, -5, "failed to generate backup codes")
		return
	}

	cfg.Enabled = true
	cfg.BackupCodes = hashed
	if err := saveTotpConfigLocked(cfg); err != nil {
		log.Errorf("failed to enable totp: %s", err)
		rsp.ErrRsp(c, -6, "failed to save config")
		return
	}

	Audit(c, "totp_enable", nil)

	rsp.OkRspWithData(c, &proto.EnableTotpRsp{BackupCodes: plain})
}

// DisableTotp requires the account password and a valid code, so a stolen
// session alone cannot strip the second factor.
func (s *Service) DisableTotp(c *gin.Context) {
	var req proto.DisableTotpReq
	var rsp proto.Response

	// Bound directly: the shared parser logs whole request bodies in debug
	// mode and this one carries the password.
	if err := c.ShouldBind(&req); err != nil || req.Password == "" || req.Code == "" {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	account, err := GetAccount()
	if err != nil {
		rsp.ErrRsp(c, -2, "failed to read account")
		return
	}
	if !CompareAccount(account.Username, req.Password) {
		AuditFailure(c, "totp_disable", "invalid password")
		rsp.ErrRsp(c, -3, "invalid password")
		return
	}
	if !VerifyTotp(req.Code) {
		AuditFailure(c, "totp_disable", "invalid code")
		rsp.ErrRsp(c, -4, "invalid code")
		return
	}

	totpMu.Lock()
	err = os.Remove(TotpConfigFile)
	totpMu.Unlock()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Errorf("failed to remove totp config: %s", err)
		rsp.ErrRsp(c, -5, "failed to disable")
		return
	}

	Audit(c, "totp_disable", nil)
	rsp.OkRsp(c)
}

func generateBackupCodes() (plain []string, hashed []string, err error) {
	for i := 0; i < backupCodeQty; i++ {
		buf := make([]byte, 6)
		if _, err = rand.Read(buf); err != nil {
			return nil, nil, fmt.Errorf("read random: %w", err)
		}
		code := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf))

		digest, hashErr := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, nil, fmt.Errorf("hash backup code: %w", hashErr)
		}

		plain = append(plain, code)
		hashed = append(hashed, string(digest))
	}

	return plain, hashed, nil
}

func loadTotpConfig() (*TotpConfig, error) {
	totpMu.Lock()
	defer totpMu.Unlock()
	return loadTotpConfigLocked()
}

func loadTotpConfigLocked() (*TotpConfig, error) {
	content, err := os.ReadFile(TotpConfigFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &TotpConfig{}, err
		}
		log.Errorf("failed to read totp config: %s", err)
		return &TotpConfig{}, errTotpUnreadable
	}

	var cfg TotpConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		log.Errorf("failed to parse totp config: %s", err)
		return &TotpConfig{}, errTotpUnreadable
	}

	return &cfg, nil
}

func saveTotpConfigLocked(cfg *TotpConfig) error {
	dir := filepath.Dir(TotpConfigFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create totp config directory: %w", err)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal totp config: %w", err)
	}

	// Written via a temporary file at 0600: it holds the shared secret, which
	// is equivalent to the second factor itself.
	tmp, err := os.CreateTemp(dir, ".totp.json.*")
	if err != nil {
		return fmt.Errorf("create temporary totp config: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set totp config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write totp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync totp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close totp config: %w", err)
	}
	if err := os.Rename(tmpPath, TotpConfigFile); err != nil {
		return fmt.Errorf("replace totp config: %w", err)
	}

	if handle, err := os.Open(dir); err == nil {
		_ = handle.Sync()
		_ = handle.Close()
	}

	return nil
}
