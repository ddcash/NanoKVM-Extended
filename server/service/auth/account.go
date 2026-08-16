package auth

import (
	"NanoKVM-Server/utils"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const AccountFile = "/etc/kvm/pwd"

type Account struct {
	Username string `json:"username"`
	Password string `json:"password"`	// should be named HashedPassword for clarity
}

func GetAccount() (*Account, error) {
	if _, err := os.Stat(AccountFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return getDefaultAccount(), nil
		}
		return nil, err
	}

	content, err := os.ReadFile(AccountFile)
	if err != nil {
		return nil, err
	}

	var account Account
	if err = json.Unmarshal(content, &account); err != nil {
		log.Errorf("unmarshal account failed: %s", err)
		return nil, err
	}

	return &account, nil
}

func SetAccount(username string, hashedPassword string) error {
	account, err := json.Marshal(&Account{
		Username: username,
		Password: hashedPassword,
	})
	if err != nil {
		log.Errorf("failed to marshal account information to json: %s", err)
		return err
	}

	// 0o644 on a directory leaves it non-traversable: a directory needs the
	// execute bit to be entered at all.
	err = os.MkdirAll(filepath.Dir(AccountFile), 0o755)
	if err != nil {
		log.Errorf("create directory %s failed: %s", AccountFile, err)
		return err
	}

	// Holds the bcrypt hash of the admin password, so it must not be
	// world-readable.
	err = os.WriteFile(AccountFile, account, 0o600)
	if err != nil {
		log.Errorf("write password failed: %s", err)
		return err
	}
	// WriteFile does not change the mode of a file that already exists, so an
	// account written by an earlier build keeps its 0o644 until tightened here.
	if err := os.Chmod(AccountFile, 0o600); err != nil {
		log.Errorf("tighten password file permissions failed: %s", err)
		return err
	}

	return nil
}

func CompareAccount(username string, plainPassword string) bool {
	account, err := GetAccount()
	if err != nil {
		return false
	}

	if username != account.Username {
		return false
	}

	hashedPassword, err := utils.DecodeDecrypt(plainPassword)
	if err != nil || hashedPassword == "" {
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(hashedPassword))
	if err != nil {
		// Compatible with old versions
		accountHashedPassword, _ := utils.DecodeDecrypt(account.Password)
		if accountHashedPassword == hashedPassword {
			return true
		}

		return false
	}

	return true
}

func DelAccount() error {
	if err := os.Remove(AccountFile); err != nil {
		log.Errorf("failed to delete password: %s", err)
		return err
	}

	return nil
}

func getDefaultAccount() *Account {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)

	return &Account{
		Username: "admin",
		Password: string(hashedPassword),
	}
}
