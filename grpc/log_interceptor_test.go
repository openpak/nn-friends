package grpc

import (
	"bytes"
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLogInterceptorOneLineNoPayload(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() { log.SetOutput(nil); log.SetFlags(log.LstdFlags) }()

	info := &grpc.UnaryServerInfo{FullMethod: "/friends.Friends/GetUserData"}
	_, _ = logInterceptor(context.Background(), "PAYLOAD-SECRET", info, func(context.Context, interface{}) (interface{}, error) {
		return "RESPONSE-SECRET", nil
	})
	_, _ = logInterceptor(context.Background(), nil, info, func(context.Context, interface{}) (interface{}, error) {
		return nil, status.Error(codes.NotFound, "no user 1234")
	})
	_, _ = logInterceptor(context.Background(), nil, info, func(context.Context, interface{}) (interface{}, error) {
		return nil, errors.New("plain")
	})
	out := buf.String()
	re := regexp.MustCompile(`(?m)^grpc /friends.Friends/GetUserData -> (OK|NotFound|Unknown) \d+ms$`)
	if got := len(re.FindAllString(out, -1)); got != 3 {
		t.Fatalf("want 3 lines, got %d: %q", got, out)
	}
	for _, s := range []string{"SECRET", "no user"} {
		if strings.Contains(out, s) {
			t.Fatalf("log leaks %q: %q", s, out)
		}
	}
}
