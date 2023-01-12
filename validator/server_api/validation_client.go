package server_api

import (
	"context"
	"errors"

	"github.com/offchainlabs/nitro/validator"

	"github.com/offchainlabs/nitro/util/stopwaiter"

	"github.com/offchainlabs/nitro/validator/server_common"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type ValidationClient struct {
	stopwaiter.StopWaiter
	client    *rpc.Client
	url       string
	name      string
	jwtSecret []byte
}

func NewValidationClient(url string, jwtSecret []byte) *ValidationClient {
	return &ValidationClient{
		url:       url,
		jwtSecret: jwtSecret,
	}
}

func (c *ValidationClient) Launch(entry *validator.ValidationInput, moduleRoot common.Hash) validator.ValidationRun {
	valrun := server_common.NewValRun(moduleRoot)
	c.LaunchThread(func(ctx context.Context) {
		input := ValidationInputToJson(entry)
		var res validator.GoGlobalState
		err := c.client.CallContext(ctx, &res, Namespace+"_validate", input, moduleRoot)
		valrun.ConsumeResult(res, err)
	})
	return valrun
}

func (c *ValidationClient) Start(ctx_in context.Context) error {
	c.StopWaiter.Start(ctx_in, c)
	ctx := c.GetContext()
	var client *rpc.Client
	var err error
	if len(c.jwtSecret) == 0 {
		client, err = rpc.DialWebsocket(ctx, c.url, "")
	} else {
		client, err = rpc.DialWebsocketJWT(ctx, c.url, "", c.jwtSecret)
	}
	if err != nil {
		return err
	}
	var name string
	err = client.CallContext(ctx, &name, Namespace+"_name")
	if err != nil {
		return err
	}
	if len(name) == 0 {
		return errors.New("couldn't read name from server")
	}
	c.client = client
	c.name = name + " on " + c.url
	return nil
}

func (c *ValidationClient) Stop() {
	c.StopWaiter.StopOnly()
	c.client.Close()
}

func (c *ValidationClient) Name() string {
	if c.Started() {
		return c.name
	}
	return "(not started) on " + c.url
}

func (c *ValidationClient) Room() int {
	var res int
	err := c.client.CallContext(c.GetContext(), &res, Namespace+"_room")
	if err != nil {
		log.Error("error contacting validation server", "name", c.name, "err", err)
		return 0
	}
	return res
}
