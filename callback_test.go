package xpaymentsdk

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestCallbackHandlerAnswersVerificationChallengeWithoutCallingBusinessHandler(t *testing.T) {
	const secret = "callback-secret"
	now := time.Now()
	body := []byte(`{"eventType":"payment.callback.test","eventId":"event-1","challenge":"challenge-1","timestamp":1}`)
	request := httptest.NewRequest(http.MethodPost, "/callback", bytes.NewReader(body))
	timestamp := strconv.FormatInt(now.Unix(), 10)
	request.Header.Set(CallbackTimestampHeader, timestamp)
	request.Header.Set(CallbackSignatureHeader, callbackSignature(secret, timestamp, body))
	called := false
	recorder := httptest.NewRecorder()

	NewCallbackHandler(secret, func(http.ResponseWriter, *http.Request, []byte) {
		called = true
	}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if called {
		t.Fatal("verification event must not reach the business payment callback")
	}
	var response callbackChallengeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Protocol != CallbackProtocol || response.Challenge != "challenge-1" || response.Proof != callbackProof(secret, "challenge-1") {
		t.Fatalf("unexpected verification response: %+v", response)
	}
}

func TestVerifyCallbackSignatureRejectsWrongOrStaleSignature(t *testing.T) {
	const secret = "callback-secret"
	now := time.Now()
	body := []byte(`{"eventType":"payment.paid"}`)
	timestamp := strconv.FormatInt(now.Unix(), 10)
	if err := VerifyCallbackSignature(secret, timestamp, callbackSignature(secret, timestamp, body), body, now, time.Minute); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if err := VerifyCallbackSignature(secret, timestamp, "00", body, now, time.Minute); err == nil {
		t.Fatal("wrong signature must be rejected")
	}
	stale := strconv.FormatInt(now.Add(-2*time.Minute).Unix(), 10)
	if err := VerifyCallbackSignature(secret, stale, callbackSignature(secret, stale, body), body, now, time.Minute); err == nil {
		t.Fatal("stale signature must be rejected")
	}
}
