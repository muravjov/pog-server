package grpcproxy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"git.catbo.net/muravjov/go2023/util"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
)

func isAuthenticated(authorization []string, authLst []AuthItem) (err error) {
	if len(authorization) < 1 {
		return errors.New("received empty authorization token from client")
	}
	tokenBase64 := strings.TrimPrefix(authorization[0], "Basic ")

	b, err := base64.StdEncoding.DecodeString(tokenBase64)
	if err != nil {
		return fmt.Errorf("base64 decoding of received token %q: %v", tokenBase64, err)
	}

	creds := string(b)
	i := strings.Index(creds, ":")
	if i < 0 {
		return fmt.Errorf("token %q misses ':' for the formatting user:password", tokenBase64)
	}
	user := creds[:i]
	pass := creds[i+1:]

	for _, aui := range authLst {
		if aui.Name != user {
			continue
		}

		if ok := doPasswordsMatch(aui.Hash, pass); !ok {
			return fmt.Errorf("wrong user and/or password")
		}

		if aui.ExpDate.Before(time.Now()) {
			return fmt.Errorf("expired user account")
		}

		return nil
	}

	return fmt.Errorf("wrong user and/or password")
}

type AuthInterceptor struct {
	AuthLst []AuthItem
}

func doAuth(ctx context.Context, authLst []AuthItem) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errMissingMetadata
	}
	err := isAuthenticated(md["authorization"], authLst)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	return nil
}

func (ai *AuthInterceptor) ProcessUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if err := doAuth(ctx, ai.AuthLst); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func (ai *AuthInterceptor) ProcessStream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := doAuth(ss.Context(), ai.AuthLst); err != nil {
		return err
	}

	return handler(srv, ss)
}

type AuthItem struct {
	Name string `json:"name"`
	Hash string `json:"hash"`

	ExpDateStr string    `json:"exp_date"`
	ExpDate    time.Time `json:"-"`
}

func hashPassword(password string) (string, error) {
	// bcrypt.MaxCost lasts too long
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashedPasswordBytes), err
}

func doPasswordsMatch(hashedPassword, currPassword string) bool {
	err := bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword), []byte(currPassword))
	return err == nil
}

func ParseAuthList() []AuthItem {
	lst := []AuthItem{}
	for _, e := range os.Environ() {
		i := strings.Index(e, "=")
		if i < 0 {
			continue
		}

		key := e[:i]
		value := e[i+1:]

		if !strings.HasPrefix(key, "POG_AUTH_") {
			continue
		}

		var ai AuthItem
		if err := json.Unmarshal([]byte(value), &ai); err != nil {
			util.Errorf("failed to parse %v auth item: %v", key, err)
			continue
		}

		expDate, err := time.Parse(time.RFC3339, ai.ExpDateStr)
		if err != nil {
			util.Errorf("failed to parse exp_date for %v auth item: %v", key, err)
			continue
		}

		ai.ExpDate = expDate

		lst = append(lst, ai)
	}

	return lst
}

type BasicAuthCredentials struct {
	Auth string
}

// GetRequestMetadata gets the request metadata as a map from a TokenSource.
func (ts BasicAuthCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	s := base64.StdEncoding.EncodeToString([]byte(ts.Auth))
	return map[string]string{
		"authorization": "Basic " + s,
	}, nil
}

func (ts BasicAuthCredentials) RequireTransportSecurity() bool {
	return false
}
