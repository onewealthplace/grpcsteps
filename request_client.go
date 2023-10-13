package grpcsteps

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"reflect"
	"sync"
	"time"

	"github.com/onewealthplace/bsonpb/v2"
	"go.nhat.io/grpcmock"
	"go.nhat.io/grpcmock/invoker"
	"go.nhat.io/grpcmock/must"
	xreflect "go.nhat.io/grpcmock/reflect"
	"go.nhat.io/grpcmock/service"
)

// ErrNoClientRequestInContext indicates that there is no client request in context.
const ErrNoClientRequestInContext err = "no client request in context"

type clientRequest interface {
	Do(ctx context.Context) ([]byte, error)
}

type clientRequestInvoker struct {
	invoker     *invoker.Invoker
	response    []byte
	responseRaw interface{}
	responseErr error

	once sync.Once
}

func (r *clientRequestInvoker) Do(ctx context.Context) ([]byte, error) {
	r.once.Do(func() {
		r.responseErr = r.invoker.Invoke(metadata.NewOutgoingContext(context.Background(), mdFromContext(ctx)))
		if r.responseErr != nil {
			return
		}

		var obj, err = bsonpb.MarshalOptions{}.Marshal(r.responseRaw.(proto.Message))
		must.NotFail(err) // this should not happen
		payload, err := bson.MarshalExtJSON(obj, true, false)

		//payload, err := assertjson.MarshalIndentCompact(&obj, "", "  ", 80)
		must.NotFail(err) // this should not happen

		r.response = payload
	})

	return r.response, r.responseErr
}

func newClientRequestInvoker(svc *Service, payload interface{}) *clientRequestInvoker {
	out := newServerOutput(svc.MethodType, svc.Output)
	i := invoker.New(svc.Method, clientRequestInvokerOptions(svc, payload, out)...)

	i.WithTimeout(time.Second)

	return &clientRequestInvoker{
		invoker:     i,
		responseRaw: out,
		responseErr: nil,
	}
}

func clientRequestInvokerOptions(svc *Service, payload interface{}, out interface{}) []invoker.Option {
	opts := []invoker.Option{
		invoker.WithAddress(svc.Address),
	}

	switch svc.MethodType {
	case service.TypeBidirectionalStream:
		opts = append(opts, invoker.WithBidirectionalStreamHandler(grpcmock.SendAndRecvAll(payload, out)))

	case service.TypeClientStream:
		opts = append(opts, invoker.WithInputStreamHandler(grpcmock.SendAll(payload)),
			invoker.WithOutput(out),
		)

	case service.TypeServerStream:
		opts = append(opts, invoker.WithInput(payload),
			invoker.WithOutputStreamHandler(grpcmock.RecvAll(out)),
		)

	case service.TypeUnary:
		fallthrough
	default:
		opts = append(opts, invoker.WithInput(payload),
			invoker.WithOutput(out),
		)
	}

	opts = append(opts, invoker.WithDialOptions(svc.DialOptions...))

	return opts
}

type missingClientRequest struct{}

func (m missingClientRequest) Do(ctx context.Context) ([]byte, error) {
	return nil, missingClientRequestPlannerErr()
}

func clientRequestFromContext(ctx context.Context) clientRequest {
	r, ok := ctx.Value(requestCtxKey{}).(clientRequest)
	if !ok {
		return missingClientRequest{}
	}

	return r
}

func mdFromContext(ctx context.Context) metadata.MD {
	r, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return metadata.MD{}
	}

	return r
}

func clientRequestToContext(ctx context.Context, r clientRequest) context.Context {
	return context.WithValue(ctx, requestCtxKey{}, r)
}

type clientRequestPlanner struct {
	request *clientRequestInvoker
}

func (c clientRequestPlanner) WithHeader(header string, value interface{}) error {
	c.request.invoker.WithInvokeOption(grpcmock.WithHeader(header, value.(string)))

	return nil
}

func (c clientRequestPlanner) WithTimeout(d time.Duration) error {
	c.request.invoker.WithTimeout(d)

	return nil
}

func newClientRequestPlanner(req *clientRequestInvoker) *clientRequestPlanner {
	return &clientRequestPlanner{
		request: req,
	}
}

func newClientRequestPlannerContext(ctx context.Context, svc *Service, payload interface{}) context.Context {
	r := newClientRequestInvoker(svc, payload)

	ctx = requestPlannerToContext(ctx, newClientRequestPlanner(r))

	return clientRequestToContext(ctx, r)
}

func newServerOutput(methodType service.Type, out interface{}) interface{} {
	result := reflect.New(xreflect.UnwrapType(out))

	if service.IsMethodServerStream(methodType) ||
		service.IsMethodBidirectionalStream(methodType) {
		value := reflect.MakeSlice(reflect.SliceOf(result.Type()), 0, 0)
		result = reflect.New(value.Type())

		result.Elem().Set(value)
	}

	return result.Interface()
}

func missingClientRequestPlannerErr() error {
	//goland:noinspection GoErrorStringFormat
	return fmt.Errorf(
		"%w, did you forget to setup a gprc request in the scenario?\n\nFor example:\n%s",
		ErrNoClientRequestInContext,
		`
        When I request a gRPC method "/grpctest.ItemService/GetItem" with payload:
        """
        {
            "id": 42
        }
        """
`,
	)
}
