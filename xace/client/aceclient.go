package client

import (
	"context"
	"fmt"
	"sync"
    "time"

    "xace/protocol"
    "xace/log"
)

var sync_aceclient sync.Once
var aceclient   *AceClient

type AceClient struct {
	xclients   map[string]XClient    //   map[interface]XClient   map[proxy.interface]XClient
	mu       sync.RWMutex

	failMode   FailMode
	selectMode SelectMode
	discovery  ServiceDiscovery
	option     Option

	selectors map[string]Selector
	Plugins   PluginContainer
	latitude  float64
	longitude float64
	auth      string

	msgchan chan *protocol.Message
}

func GetAceClient() *AceClient {
    sync_aceclient.Do(func() {
        aceclient = NewAceClient()
    })
    return aceclient
}

// NewAceClient creates a AceClient that supports service discovery and service governance.
func NewAceClient() *AceClient {
    discovery,_ := NewAceDiscovery("", "", 5*time.Second)
    client := &AceClient{
		failMode:   Failbackup,//Failfast,
		selectMode: RandomSelect,
		discovery:  discovery,
		option:     AceOption,
		xclients:   make(map[string]XClient),
		selectors:  make(map[string]Selector),
        msgchan:    make(chan *protocol.Message, 100),
	}
    go client.handle()
    return client
}

func (c *AceClient) handle() {
    for msg :=range c.msgchan {
        log.Debugf("aceclient recv packet. %s:%s", msg.ServicePath,msg.ServiceMethod)
    }
}

// SetSelector sets customized selector by users.
func (c *AceClient) SetSelector(servicePath string, s Selector) {
	c.mu.Lock()
	c.selectors[servicePath] = s
	if xclient, ok := c.xclients[servicePath]; ok {
		xclient.SetSelector(s)
	}
	c.mu.Unlock()
}

// SetPlugins sets client's plugins.
func (c *AceClient) SetPlugins(plugins PluginContainer) {
	c.Plugins = plugins
	c.mu.RLock()
	for _, v := range c.xclients {
		v.SetPlugins(plugins)
	}
	c.mu.RUnlock()
}

func (c *AceClient) GetPlugins() PluginContainer {
	return c.Plugins
}

func (c *AceClient) newXClient(servicePath string) (xclient XClient, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	d, err := c.discovery.Clone(servicePath)
	if err != nil {
		return nil, err
	}

    _,inter := splitAcePath(servicePath)
	if c.msgchan == nil {
		xclient = NewXClient(inter, c.failMode, c.selectMode, d, c.option)
	} else {
		xclient = NewBidirectionalXClient(inter, c.failMode, c.selectMode, d, c.option, c.msgchan)
	}

	if c.Plugins != nil {
		xclient.SetPlugins(c.Plugins)
	}

	if s, ok := c.selectors[servicePath]; ok {
		xclient.SetSelector(s)
	}

	if c.selectMode == Closest {
		xclient.ConfigGeoSelector(c.latitude, c.longitude)
	}

	if c.auth != "" {
		xclient.Auth(c.auth)
	}

	return xclient, err
}

func (c *AceClient) getXClient(servicePath string) (xclient XClient, err error)  {
	c.mu.RLock()
	xclient = c.xclients[servicePath]
	c.mu.RUnlock()

	if xclient == nil {
		c.mu.Lock()
		xclient = c.xclients[servicePath]
		if xclient == nil {
			xclient, err = c.newXClient(servicePath)
			c.xclients[servicePath] = xclient
		}
		c.mu.Unlock()
	}
    return xclient, err
}

func (c *AceClient) GetXClient(servicePath string) (xclient XClient)  {
    xclient,_ = c.getXClient(servicePath)
    return xclient
}

func (c *AceClient) Go(ctx context.Context, servicePath string, serviceMethod string, args interface{}, reply interface{}, done chan *Call) (*Call, error) {
    xclient, err := c.getXClient(servicePath)
    if err != nil {
        return nil, err
    }
	return xclient.Go(ctx, serviceMethod, args, reply, done)
}

func (c *AceClient) Call(ctx context.Context, servicePath string, serviceMethod string, args interface{}, reply interface{}) error {
    xclient, err := c.getXClient(servicePath)
    if err != nil {
        return err
    }
	return xclient.Call(ctx, serviceMethod, args, reply)
}

func (c *AceClient) SendRaw(ctx context.Context, r *protocol.Message) (map[string]string, []byte, error) {
    xclient, err := c.getXClient(r.ServicePath)
    if err != nil {
        return nil, nil, err
    }
	return xclient.SendRaw(ctx, r)
}

func (c *AceClient) Broadcast(ctx context.Context, servicePath string, serviceMethod string, args interface{}, reply interface{}) error {
    xclient, err := c.getXClient(servicePath)
    if err != nil {
        return err
    }
	return xclient.Broadcast(ctx, serviceMethod, args, reply)
}

func (c *AceClient) Fork(ctx context.Context, servicePath string, serviceMethod string, args interface{}, reply interface{}) error {
    xclient, err := c.getXClient(servicePath)
    if err != nil {
        return err
    }
	return xclient.Fork(ctx, serviceMethod, args, reply)
}

func (c *AceClient) Close() error {
	var result error

	c.mu.RLock()
	for _, v := range c.xclients {
		err := v.Close()
		if err != nil {
			result = err //multierror.Append(result, err)
		}
	}
	c.mu.RUnlock()

	return result
}
