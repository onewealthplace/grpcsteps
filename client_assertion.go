package grpcsteps

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/godogx/vars"
	"github.com/swaggest/assertjson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func assertServerResponsePayloadEqual(ctx context.Context, req clientRequest, expected string) error {
	actual, err := req.Do(ctx)
	if err != nil {
		return fmt.Errorf("an error occurred while send grpc request: %w", err)
	}
	var doc bson.M
	_ = bson.UnmarshalExtJSON([]byte(actual), true, &doc)
	vars.ToContext(ctx, "$response", doc)
	return assertjson.FailNotEqual([]byte(expected), actual)
}

func assertServerResponsePayloadMatch(ctx context.Context, req clientRequest, expected string) error {
	actual, err := req.Do(ctx)
	if err != nil {
		return fmt.Errorf("an error occurred while send grpc request: %w", err)
	}
	var doc bson.M
	_ = bson.UnmarshalExtJSON([]byte(actual), true, &doc)
	vars.ToContext(ctx, "$response", doc)
	return assertjson.FailMismatch([]byte(expected), actual)
}

func assertServerResponseErrorCode(ctx context.Context, req clientRequest, expected codes.Code) error {
	_, err := req.Do(ctx)
	if err == nil {
		if expected != codes.OK {
			return fmt.Errorf("got no error, want %q", expected) // nolint: goerr113
		}

		return nil
	}

	actual := status.Convert(err).Code()

	if expected != actual {
		return fmt.Errorf("got %w, want %q", err, expected)
	}

	return nil
}

func assertServerResponseErrorMessage(ctx context.Context, req clientRequest, expected string) error {
	_, err := req.Do(ctx)
	if err == nil {
		if expected != "" {
			return fmt.Errorf("got no error, want %q", expected) // nolint: goerr113
		}

		return nil
	}

	actual := status.Convert(err).Message()

	if expected != actual {
		return fmt.Errorf("unexpected error message, got %q, want %q", actual, expected) // nolint: goerr113
	}

	return nil
}
