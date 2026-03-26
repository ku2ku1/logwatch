package auth

import (
	"database/sql"
	"encoding/base64"
	"bytes"
	"image/png"
	"fmt"

	"github.com/pquerna/otp/totp"
)

type TOTPManager struct {
	db *sql.DB
}

func NewTOTPManager(db *sql.DB) *TOTPManager {
	return &TOTPManager{db: db}
}

func (t *TOTPManager) Init() error {
	_, err := t.db.Exec(`
		CREATE TABLE IF NOT EXISTS user_totp (
			user_id   INTEGER PRIMARY KEY,
			secret    TEXT NOT NULL,
			enabled   BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`)
	return err
}

type TOTPSetup struct {
	Secret  string `json:"secret"`
	QRCode  string `json:"qr_code"` // base64 PNG
	Issuer  string `json:"issuer"`
	Account string `json:"account"`
}

func (t *TOTPManager) Generate(userID int64, username string) (*TOTPSetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "LogWatch",
		AccountName: username,
		Period:      30,
		Digits:      6,
	})
	if err != nil {
		return nil, err
	}

	// Save secret (not enabled yet)
	_, err = t.db.Exec(`
		INSERT INTO user_totp (user_id, secret, enabled)
		VALUES ($1, $2, false)
		ON CONFLICT (user_id) DO UPDATE SET secret = excluded.secret, enabled = false
	`, userID, key.Secret())
	if err != nil {
		return nil, err
	}

	// Generate QR code
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	qrBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	return &TOTPSetup{
		Secret:  key.Secret(),
		QRCode:  qrBase64,
		Issuer:  "LogWatch",
		Account: username,
	}, nil
}

func (t *TOTPManager) Verify(userID int64, code string) (bool, error) {
	var secret string
	err := t.db.QueryRow(`SELECT secret FROM user_totp WHERE user_id = $1`, userID).Scan(&secret)
	if err != nil {
		return false, err
	}
	return totp.Validate(code, secret), nil
}

func (t *TOTPManager) Enable(userID int64, code string) error {
	valid, err := t.Verify(userID, code)
	if err != nil || !valid {
		return fmt.Errorf("invalid code")
	}
	_, err = t.db.Exec(`UPDATE user_totp SET enabled = true WHERE user_id = $1`, userID)
	return err
}

func (t *TOTPManager) IsEnabled(userID int64) bool {
	var enabled bool
	err := t.db.QueryRow(`SELECT enabled FROM user_totp WHERE user_id = $1`, userID).Scan(&enabled)
	if err != nil {
		return false
	}
	return enabled
}

func (t *TOTPManager) Disable(userID int64) error {
	_, err := t.db.Exec(`DELETE FROM user_totp WHERE user_id = $1`, userID)
	return err
}
