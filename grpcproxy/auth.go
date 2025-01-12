package grpcproxy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"git.catbo.net/muravjov/go2023/util"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	errMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
)

func isAuthenticated(authorization string, authLst []AuthItem) (string, error) {
	tokenBase64 := strings.TrimPrefix(authorization, "Basic ")

	b, err := base64.StdEncoding.DecodeString(tokenBase64)
	if err != nil {
		return "", fmt.Errorf("base64 decoding of received token %q: %v", tokenBase64, err)
	}

	creds := string(b)
	i := strings.Index(creds, ":")
	if i < 0 {
		return "", fmt.Errorf("token %q misses ':' for the formatting user:password", tokenBase64)
	}
	user := creds[:i]
	pass := creds[i+1:]

	for _, aui := range authLst {
		if aui.Name != user {
			continue
		}

		if ok := doPasswordsMatch(aui.Hash, pass); !ok {
			return "", fmt.Errorf("wrong user and/or password")
		}

		if aui.ExpDate.Before(time.Now()) {
			return "", fmt.Errorf("expired user account")
		}

		return aui.Name, nil
	}

	return "", fmt.Errorf("wrong user and/or password")
}

type AuthInterceptor struct {
	AuthLst []AuthItem
}

func doAuth(ctx context.Context, authLst []AuthItem) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errMissingMetadata
	}

	authorization := md["authorization"]
	if len(authorization) < 1 {
		return "", status.Error(codes.Unauthenticated, "received empty authorization token from client")
	}

	user, err := isAuthenticated(authorization[0], authLst)
	if err != nil {
		return user, status.Error(codes.Unauthenticated, err.Error())
	}

	return user, nil
}

func (ai *AuthInterceptor) ProcessUnary(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if _, err := doAuth(ctx, ai.AuthLst); err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

func newWrappedStream(ctx context.Context, s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s, ctx}
}

type ConnectionAuthCtx struct {
	User string
}

type connectionAuthKey struct{}

func (ai *AuthInterceptor) ProcessStream(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()
	user, err := doAuth(ctx, ai.AuthLst)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, connectionAuthKey{}, ConnectionAuthCtx{
		User: user,
	})

	return handler(srv, newWrappedStream(ctx, ss))
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

	if err != nil && err != bcrypt.ErrMismatchedHashAndPassword {
		// godotenv utility expands values of env variable if
		// they are not enclosed into '' or "" => bcrypt values are being cut badly
		util.Errorf("Oops (invalid hash?): err='%v', hash='%v', pass='%v'", err, hashedPassword, "***")
	}

	return err == nil
}

const POGAuthEnvVarPrefix = "POG_AUTH_"
const ClientAuthEnvVarPrefix = "CLIENT_AUTH_"

func ParseAuthList(envVarPrefix string) ([]AuthItem, error) {
	lst := []AuthItem{}
	for _, e := range os.Environ() {
		i := strings.Index(e, "=")
		if i < 0 {
			continue
		}

		key := e[:i]
		value := e[i+1:]

		if !strings.HasPrefix(key, envVarPrefix) {
			continue
		}

		var ai AuthItem
		if err := json.Unmarshal([]byte(value), &ai); err != nil {
			err = fmt.Errorf("failed to parse %v auth item: %v", key, err)
			util.Error(err)
			return nil, err
		}

		expDate, err := time.Parse(time.RFC3339, ai.ExpDateStr)
		if err != nil {
			err = fmt.Errorf("failed to parse exp_date for %v auth item: %v", key, err)
			util.Error(err)
			return nil, err
		}

		ai.ExpDate = expDate

		lst = append(lst, ai)
	}

	if len(lst) > 0 {
		var earliestExpire time.Time
		for _, item := range lst {
			if earliestExpire.IsZero() || item.ExpDate.Before(earliestExpire) {
				earliestExpire = item.ExpDate
			}
		}

		authItemEarliestExpire.With(prometheus.Labels{"name": envVarPrefix}).Set(float64(earliestExpire.Unix()))
	}

	return lst, nil
}

var authItemEarliestExpire = util.NewGaugeVecMetric(
	"auth_item_earliest_expiry",
	"Returns earliest auth item expiry in unixtime",
	[]string{"name"},
)

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

func GenAuthItem(name string, pass string, timeToLive time.Duration) string {
	hash, _ := hashPassword(pass)

	expirationDate := time.Now().UTC().Add(timeToLive)

	b, _ := json.Marshal(AuthItem{
		Name:       name,
		Hash:       hash,
		ExpDateStr: expirationDate.Format(time.RFC3339),
	})
	fmt.Println(string(b))

	return hash
}
