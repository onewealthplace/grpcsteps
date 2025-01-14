package grpcsteps

import (
	"context"
	"github.com/godogx/vars"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	xreflect "go.nhat.io/grpcmock/reflect"
	"go.nhat.io/grpcmock/service"
	"google.golang.org/grpc"
)

// Service contains needed information to form a GRPC request.
type Service struct {
	service.Method

	Address     string
	DialOptions []grpc.DialOption
}

// ServiceOption sets up a service.
type ServiceOption func(s *Service)

// Client is a grpc client for godog.
type Client struct {
	services          map[string]*Service
	servicePrefix     string
	defaultSvcOptions []ServiceOption
}

// ClientOption sets up a client.
type ClientOption func(s *Client)

func (c *Client) registerService(id string, svc interface{}, opts ...ServiceOption) {
	for _, method := range xreflect.FindServiceMethods(svc) {
		svc := &Service{
			Method: service.Method{
				ServiceName: id,
				MethodName:  method.Name,
				MethodType:  service.ToType(method.IsClientStream, method.IsServerStream),
				Input:       method.Input,
				Output:      method.Output,
			},
			Address: ":9090",
		}

		// Apply default options.
		for _, o := range c.defaultSvcOptions {
			o(svc)
		}

		// Apply inline options.
		for _, o := range opts {
			o(svc)
		}

		fullName := svc.FullName()
		if c.servicePrefix != "" && len(fullName) > len(c.servicePrefix) && fullName[:len(c.servicePrefix)] == c.servicePrefix {
			fullName = fullName[len(c.servicePrefix):]
		}
		c.services[fullName] = svc
	}
}

// RegisterContext registers to godog scenario.
func (c *Client) RegisterContext(sc *godog.ScenarioContext) {
	sc.Step(`^I request(?: a)? (?:gRPC|GRPC|grpc)(?: method)? "([^"]*)" with payload:?$`, c.iRequestWithPayloadFromDocString)
	sc.Step(`^I request(?: a)? (?:gRPC|GRPC|grpc)(?: method)? "([^"]*)" with payload from file "([^"]+)"$`, c.iRequestWithPayloadFromFile)
	sc.Step(`^I request(?: a)? (?:gRPC|GRPC|grpc)(?: method)? "([^"]*)" with payload from file:$`, c.iRequestWithPayloadFromFileDocString)

	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response with payload:?$`, c.iShouldHaveResponseWithPayloadFromDocString)
	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response with payload from file "([^"]+)"$`, c.iShouldHaveResponseWithPayloadFromFile)
	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response with payload from file:?$`, c.iShouldHaveResponseWithPayloadFromFileDocString)

	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response match payload:?$`, c.iShouldHaveResponseMatchPayloadFromDocString)
	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response match payload from file "([^"]+)"$`, c.iShouldHaveResponseMatchPayloadFromFile)
	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response match payload from file:?$`, c.iShouldHaveResponseMatchPayloadFromFileDocString)

	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response with code "([^"]*)"$`, c.iShouldHaveResponseWithCode)
	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response with error (?:message )?"([^"]*)"$`, c.iShouldHaveResponseWithErrorMessage)
	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response with code "([^"]*)" and error (?:message )?"([^"]*)"$`, c.iShouldHaveResponseWithCodeAndErrorMessage)
	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response with error(?: message)?:$`, c.iShouldHaveResponseWithErrorMessageFromDocString)
	sc.Step(`^I should have(?: a)? (?:gRPC|GRPC|grpc) response with code "([^"]*)" and error(?: message)?:$`, c.iShouldHaveResponseWithCodeAndErrorMessageFromDocString)

	registerRequestPlanner(sc)
}

func (c *Client) iRequestWithPayload(ctx context.Context, method string, data string) (context.Context, error) {
	svc, ok := c.services[method]
	if !ok {
		return ctx, ErrInvalidGRPCMethod
	}
	currentVars := vars.FromContext(ctx)
	for k, v := range currentVars {
		switch vv := v.(type) {
		case string:
			data = strings.ReplaceAll(data, k, vv)
		case int:
			data = strings.ReplaceAll(data, k, strconv.Itoa(vv))
		}
	}
	payload, err := toPayload(svc.MethodType, svc.Input, &data)
	if err != nil {
		return ctx, err
	}

	return newClientRequestPlannerContext(ctx, svc, payload), nil
}

func (c *Client) iRequestWithPayloadFromDocString(ctx context.Context, method string, payload *godog.DocString) (context.Context, error) {
	return c.iRequestWithPayload(ctx, method, payload.Content)
}

func (c *Client) iRequestWithPayloadFromFile(ctx context.Context, method string, path string) (context.Context, error) {
	payload, err := os.ReadFile(path) // nolint: gosec
	if err != nil {
		return ctx, err
	}

	return c.iRequestWithPayload(ctx, method, string(payload))
}

func (c *Client) iRequestWithPayloadFromFileDocString(ctx context.Context, method string, path *godog.DocString) (context.Context, error) {
	return c.iRequestWithPayloadFromFile(ctx, method, path.Content)
}

func (c *Client) iShouldHaveResponseWithPayload(ctx context.Context, response string) error {
	return assertServerResponsePayloadEqual(ctx, clientRequestFromContext(ctx), response)
}

func (c *Client) iShouldHaveResponseWithPayloadFromDocString(ctx context.Context, response *godog.DocString) error {
	return c.iShouldHaveResponseWithPayload(ctx, response.Content)
}

func (c *Client) iShouldHaveResponseWithPayloadFromFile(ctx context.Context, path string) error {
	payload, err := os.ReadFile(path) // nolint: gosec
	if err != nil {
		return err
	}

	return c.iShouldHaveResponseWithPayload(ctx, string(payload))
}

func (c *Client) iShouldHaveResponseWithPayloadFromFileDocString(ctx context.Context, path *godog.DocString) error {
	return c.iShouldHaveResponseWithPayloadFromFile(ctx, path.Content)
}

func (c *Client) iShouldHaveResponseMatchPayload(ctx context.Context, response string) error {
	return assertServerResponsePayloadMatch(ctx, clientRequestFromContext(ctx), response)
}

func (c *Client) iShouldHaveResponseMatchPayloadFromDocString(ctx context.Context, response *godog.DocString) error {
	return c.iShouldHaveResponseMatchPayload(ctx, response.Content)
}

func (c *Client) iShouldHaveResponseMatchPayloadFromFile(ctx context.Context, path string) error {
	payload, err := os.ReadFile(path) // nolint: gosec
	if err != nil {
		return err
	}

	return c.iShouldHaveResponseMatchPayload(ctx, string(payload))
}

func (c *Client) iShouldHaveResponseMatchPayloadFromFileDocString(ctx context.Context, path *godog.DocString) error {
	return c.iShouldHaveResponseMatchPayloadFromFile(ctx, path.Content)
}

func (c *Client) iShouldHaveResponseWithCode(ctx context.Context, codeValue string) error {
	code, err := toStatusCode(codeValue)
	if err != nil {
		return err
	}

	return assertServerResponseErrorCode(ctx, clientRequestFromContext(ctx), code)
}

func (c *Client) iShouldHaveResponseWithErrorMessage(ctx context.Context, err string) error {
	return assertServerResponseErrorMessage(ctx, clientRequestFromContext(ctx), err)
}

func (c *Client) iShouldHaveResponseWithCodeAndErrorMessage(ctx context.Context, codeValue, err string) error {
	if err := c.iShouldHaveResponseWithCode(ctx, codeValue); err != nil {
		return err
	}

	return c.iShouldHaveResponseWithErrorMessage(ctx, err)
}

func (c *Client) iShouldHaveResponseWithErrorMessageFromDocString(ctx context.Context, err *godog.DocString) error {
	return c.iShouldHaveResponseWithErrorMessage(ctx, err.Content)
}

func (c *Client) iShouldHaveResponseWithCodeAndErrorMessageFromDocString(ctx context.Context, codeValue string, err *godog.DocString) error {
	if err := c.iShouldHaveResponseWithCode(ctx, codeValue); err != nil {
		return err
	}

	return c.iShouldHaveResponseWithErrorMessage(ctx, err.Content)
}

// NewClient initiates a new grpc server extension for testing.
func NewClient(opts ...ClientOption) *Client {
	s := &Client{
		services: make(map[string]*Service),
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

// RegisterServiceFromInstance registers a grpc server by its interface.
func RegisterServiceFromInstance(id string, svc interface{}, opts ...ServiceOption) ClientOption {
	return func(c *Client) {
		c.registerService(id, svc, opts...)
	}
}

// RegisterService registers a grpc server by its interface.
func RegisterService(registerFunc interface{}, opts ...ServiceOption) ClientOption {
	return func(c *Client) {
		serviceDesc, svc := xreflect.ParseRegisterFunc(registerFunc)

		c.registerService(serviceDesc.ServiceName, svc, opts...)
	}
}

// WithDefaultServiceOptions set default service options.
func WithDefaultServiceOptions(opts ...ServiceOption) ClientOption {
	return func(s *Client) {
		s.defaultSvcOptions = append(s.defaultSvcOptions, opts...)
	}
}

// AddrProvider provides a net address.
type AddrProvider interface {
	Addr() net.Addr
}

// WithAddressProvider sets service address.
func WithAddressProvider(p AddrProvider) ServiceOption {
	return WithAddr(p.Addr().String())
}

// WithAddr sets service address.
func WithAddr(addr string) ServiceOption {
	return func(s *Service) {
		s.Address = addr
	}
}

// WithDialOption adds a dial option.
func WithDialOption(o grpc.DialOption) ServiceOption {
	return func(s *Service) {
		s.DialOptions = append(s.DialOptions, o)
	}
}

// WithDialOptions sets dial options.
func WithDialOptions(opts ...grpc.DialOption) ServiceOption {
	return func(s *Service) {
		s.DialOptions = opts
	}
}

func WithServicePrefix(prefix string) ClientOption {
	return func(c *Client) {
		c.servicePrefix = prefix
	}
}
