package xpaymentsdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	CallbackProtocol        = "xpayment-callback-v1"
	CallbackSignatureHeader = "X-XPayment-Signature"
	CallbackTimestampHeader = "X-XPayment-Timestamp"
	CallbackTestEvent       = "payment.callback.test"
	defaultCallbackMaxAge   = 5 * time.Minute
)

var ErrInvalidCallbackSignature = errors.New("invalid xpayment callback signature")

type CallbackHandlerFunc func(http.ResponseWriter, *http.Request, []byte)

type callbackEnvelope struct {
	EventType string `json:"eventType"`
	Challenge string `json:"challenge"`
}

type callbackChallengeResponse struct {
	Protocol  string `json:"protocol"`
	Challenge string `json:"challenge"`
	Proof     string `json:"proof"`
}

func VerifyCallbackSignature(secret, timestamp, signature string, body []byte, now time.Time, maxAge time.Duration) error {
	return verifyCallbackSignatureHeader(secret, timestamp, signature, body, now, maxAge)
}

func NewCallbackHandler(secret string, next CallbackHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid callback body", http.StatusBadRequest)
			return
		}
		timestamp := r.Header.Get(CallbackTimestampHeader)
		if err = verifyCallbackSignatureHeader(secret, timestamp, r.Header.Get(CallbackSignatureHeader), body, time.Now(), defaultCallbackMaxAge); err != nil {
			http.Error(w, "invalid callback signature", http.StatusUnauthorized)
			return
		}
		responseBody, challenge, challengeErr := CallbackChallengeResponse(secret, body)
		if challengeErr != nil {
			http.Error(w, "invalid callback payload", http.StatusBadRequest)
			return
		}
		if challenge {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(responseBody)
			return
		}
		if next == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r, body)
	})
}

func CallbackChallengeResponse(secret string, body []byte) ([]byte, bool, error) {
	var envelope callbackEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, false, err
	}
	if envelope.EventType != CallbackTestEvent {
		return nil, false, nil
	}
	if strings.TrimSpace(envelope.Challenge) == "" {
		return nil, true, errors.New("xpayment callback challenge is required")
	}
	response := callbackChallengeResponse{
		Protocol: CallbackProtocol, Challenge: envelope.Challenge,
		Proof: callbackProof(secret, envelope.Challenge),
	}
	raw, err := json.Marshal(response)
	return raw, true, err
}

func verifyCallbackSignatureHeader(secret, timestamp, signature string, body []byte, now time.Time, maxAge time.Duration) error {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(timestamp) == "" || strings.TrimSpace(signature) == "" {
		return ErrInvalidCallbackSignature
	}
	unixSeconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrInvalidCallbackSignature
	}
	if maxAge <= 0 {
		maxAge = defaultCallbackMaxAge
	}
	signedAt := time.Unix(unixSeconds, 0)
	if now.Sub(signedAt) > maxAge || signedAt.Sub(now) > maxAge {
		return ErrInvalidCallbackSignature
	}
	expected, err := hex.DecodeString(callbackSignature(secret, timestamp, body))
	if err != nil {
		return ErrInvalidCallbackSignature
	}
	provided, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil || !hmac.Equal(provided, expected) {
		return ErrInvalidCallbackSignature
	}
	return nil
}

func callbackSignature(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func callbackProof(secret, challenge string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(CallbackProtocol + "." + challenge))
	return hex.EncodeToString(mac.Sum(nil))
}
