package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	ddhttp "github.com/unionj-cloud/go-doudou/svc/http"
)

type UserClient struct {
	provider ddhttp.IServiceProvider
	client   *resty.Client
}

func (receiver *UserClient) SetProvider(provider ddhttp.IServiceProvider) {
	receiver.provider = provider
}

func (receiver *UserClient) SetClient(client *resty.Client) {
	receiver.client = client
}

// Get user by user name
func (receiver *UserClient) GetUserUsername(ctx context.Context, username string) (ret User, err error) {
	var (
		_server string
		_err    error
	)
	if _server, _err = receiver.provider.SelectServer(); _err != nil {
		err = errors.Wrap(_err, "")
		return
	}

	_req := receiver.client.R()
	_req.SetContext(ctx)
	_req.SetPathParam("username", fmt.Sprintf("%v", username))

	_resp, _err := _req.Get(_server + "/user/{username}")
	if _err != nil {
		err = errors.Wrap(_err, "")
		return
	}
	if _resp.IsError() {
		err = errors.New(_resp.String())
		return
	}
	if _err = json.Unmarshal(_resp.Body(), &ret); _err != nil {
		err = errors.Wrap(_err, "")
		return
	}
	return
}

// Creates list of users with given input array
// Creates list of users with given input array
func (receiver *UserClient) PostUserCreateWithList(ctx context.Context, bodyJson []User) (ret User, err error) {
	var (
		_server string
		_err    error
	)
	if _server, _err = receiver.provider.SelectServer(); _err != nil {
		err = errors.Wrap(_err, "")
		return
	}

	_req := receiver.client.R()
	_req.SetContext(ctx)
	_req.SetBody(bodyJson)

	_resp, _err := _req.Post(_server + "/user/createWithList")
	if _err != nil {
		err = errors.Wrap(_err, "")
		return
	}
	if _resp.IsError() {
		err = errors.New(_resp.String())
		return
	}
	if _err = json.Unmarshal(_resp.Body(), &ret); _err != nil {
		err = errors.Wrap(_err, "")
		return
	}
	return
}

// Logs user into the system
func (receiver *UserClient) GetUserLogin(ctx context.Context) (ret string, err error) {
	var (
		_server string
		_err    error
	)
	if _server, _err = receiver.provider.SelectServer(); _err != nil {
		err = errors.Wrap(_err, "")
		return
	}

	_req := receiver.client.R()
	_req.SetContext(ctx)

	_resp, _err := _req.Get(_server + "/user/login")
	if _err != nil {
		err = errors.Wrap(_err, "")
		return
	}
	if _resp.IsError() {
		err = errors.New(_resp.String())
		return
	}
	if _err = json.Unmarshal(_resp.Body(), &ret); _err != nil {
		err = errors.Wrap(_err, "")
		return
	}
	return
}

func NewUser(opts ...ddhttp.DdClientOption) *UserClient {
	defaultProvider := ddhttp.NewServiceProvider("User")
	defaultClient := ddhttp.NewClient()

	svcClient := &UserClient{
		provider: defaultProvider,
		client:   defaultClient,
	}

	for _, opt := range opts {
		opt(svcClient)
	}

	return svcClient
}