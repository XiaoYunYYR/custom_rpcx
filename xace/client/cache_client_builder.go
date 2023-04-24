package client

import (
    "sync"
    "xace/log"
)

// CacheClientBuilder defines builder interface to generate RPCCient.
type CacheClientBuilder interface {
	SetCachedClient(client RPCClient, k, servicePath, serviceMethod string)
	FindCachedClient(k, servicePath, serviceMethod string) RPCClient
	DeleteCachedClient(client RPCClient, k, servicePath, serviceMethod string)
	GenerateClient(k, servicePath, serviceMethod string) (client RPCClient, err error)
}

// RegisterCacheClientBuilder(network string, builder CacheClientBuilder)

var cacheClientBuildersMutex sync.RWMutex
var cacheClientBuilders = make(map[string]CacheClientBuilder)

func RegisterCacheClientBuilder(network string, builder CacheClientBuilder) {
	cacheClientBuildersMutex.Lock()
	defer cacheClientBuildersMutex.Unlock()

	cacheClientBuilders[network] = builder
}

func getCacheClientBuilder(network string) (CacheClientBuilder, bool) {
	cacheClientBuildersMutex.RLock()
	defer cacheClientBuildersMutex.RUnlock()

	builder, ok := cacheClientBuilders[network]
	return builder, ok
}


type aceClientBuilder struct {
    mux     sync.RWMutex
    clients map[string]RPCClient
}

func InitAceClientBuilder() {
    _,ok := getCacheClientBuilder("ace")
    if !ok {
        builder := &aceClientBuilder{}
        builder.clients = make(map[string]RPCClient, 20)
        RegisterCacheClientBuilder("ace", builder)
    }
}

func (b *aceClientBuilder) SetCachedClient(client RPCClient, k, servicePath, serviceMethod string) {
    b.mux.Lock()
    defer b.mux.Unlock()
    b.clients[k] = client
}

func (b *aceClientBuilder) FindCachedClient(k, servicePath, serviceMethod string) RPCClient {
    b.mux.RLock()
    defer b.mux.RUnlock()
    client, _ := b.clients[k]
    return client
}

func (b *aceClientBuilder) DeleteCachedClient(client RPCClient, k, servicePath, serviceMethod string) {
    b.mux.Lock()
    defer b.mux.Unlock()
    c, ok := b.clients[k]
    if ok {
        if c == client {
            delete(b.clients,k)
            log.Infof("del cache client %s %s",k,servicePath)
        }
    }
}

func (b *aceClientBuilder) GenerateClient(k, servicePath, serviceMethod string) (client RPCClient, err error) {
    network,addr := splitNetworkAndAddress(k)

    log.Infof("gen cache client %s %s",k,servicePath)

	client = &Client{
		option:  AceOption,
		Plugins: AcePluginContainer,
	}

	err = client.Connect(network, addr)
	return client, err
}

