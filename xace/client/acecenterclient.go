package client

import (
    "time"
    "sync"
    "context"
    "xace/log"
    "xace/protocol"
)

type AceCenterSrvInfo struct {
    Proxy   string
    Host    string
    Port    string
    Status  uint8
}


type AceCenterClient struct {
    domain      string

	xclient     XClient

    msgchan chan *protocol.Message
}

var sync_center sync.Once
var centerclient *AceCenterClient

func GetAceCenterClient() *AceCenterClient {
    sync_center.Do(func() {
        centerclient = &AceCenterClient{}
        centerclient.msgchan = make(chan *protocol.Message, 10)
        go centerclient.handle()
    })
    return centerclient
}

func AceGetProxyInfos(proxy, inter string) (error, []AceCenterSrvInfo) {
    center := GetAceCenterClient()
    return center.getProxyInfos(proxy, inter)
}

func (c *AceCenterClient) handle() {
    for msg :=range c.msgchan {
        log.Debugf("acecenterclient recv packet. %s:%s", msg.ServicePath,msg.ServiceMethod)
    }
}

func (c *AceCenterClient) getProxyInfos(proxy, inter string) (error, []AceCenterSrvInfo) {
    srvs := make([]AceCenterSrvInfo, 0, 3)
    if c.xclient == nil {
        return nil, srvs
    }

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    var args []any = make([]any,0,2)
    args = append(args,proxy)
    args = append(args,inter)
    var reply []any = make([]any,0, 2)
    //srvs := make([]AceCenterSrvInfo, 0, 3)
    reply = append(reply, &srvs)
    err := c.xclient.Call(ctx, "getProxy", args, reply)
    if err != nil {
        return err, nil
    }
    return nil, srvs
}

func (c *Client) getProxyInfos(proxy, inter string) (error, []AceCenterSrvInfo) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    var args []any = make([]any,0,2)
    args = append(args,proxy)
    args = append(args,inter)
    var reply []any = make([]any,0, 2)
    srvs := make([]AceCenterSrvInfo, 0, 3)
    reply = append(reply, &srvs)
    err := c.Call(ctx, "AaceCenter", "getProxy", args, reply)
    if err != nil {
        return err, nil
    }
    return nil, srvs
}

func generateAceCenterClient(domain string, wait time.Duration) *Client {
    curtime := time.Now().UnixMilli()
    for {
        client := NewClient(AceOption)
        err := client.Connect("tcp",domain)
        if err == nil && client != nil {
            return client
        }
        if wait <= 0 || (time.Now().UnixMilli()-curtime) > wait.Milliseconds() {
            log.Error("generate center client time out. ", err)
            break
        }
        log.Warn("generate center client. ", err)
    }
    return nil
}

func InitializeAceCenter(domain string, proxy string) {
    center := GetAceCenterClient()
    if len(domain) == 0 {
        return
    }
    center.domain = domain

    client := generateAceCenterClient(domain, 0*time.Second)
    addr := client.RemoteAddr()

    InitAceClientBuilder()

    discovery,_ := NewAceDiscovery(proxy, "AaceCenter", 10*time.Second)

	center.xclient = NewBidirectionalXClient("AaceCenter", Failfast, RandomSelect, discovery, AceOption, center.msgchan)

    xclient,_ := center.xclient.(*xClient)
	xclient.setInitAceClient(client, "ace@"+addr, "AaceCenter", "")

}



