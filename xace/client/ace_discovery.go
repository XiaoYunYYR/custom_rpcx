package client

import (
    "fmt"
    "sort"
    "strings"
    "sync"
    "time"
    "xace/log"
)

type AceDiscovery struct {
    proxy       string
    inter       string
    d           time.Duration

    pairsMu     sync.RWMutex
    pairs       []*KVPair
    chans       []chan []*KVPair

    mu          sync.Mutex
    filter      ServiceDiscoveryFilter

    stopCh      chan  struct{}
}

func AceDiscoveryFilter(kvp *KVPair) bool {
    return kvp.Value == "2"
}

func NewAceDiscovery(proxy string, inter string, d time.Duration) (*AceDiscovery, error) {
    discovery := &AceDiscovery{proxy:proxy,inter:inter, d:d, filter:AceDiscoveryFilter}
    if len(proxy) > 0 {
        discovery.lookup()
        go discovery.watch()
    }
    return discovery, nil
}

func splitAcePath(path string) (proxy string, inter string) {
    pos := strings.Index(path,".")
    if pos < 0  {
        proxy, inter = path, path
    } else {
        proxy = path[:pos]
        inter = path[pos+1:]
    }
    return
}

func (d *AceDiscovery) Clone(servicePath string) (ServiceDiscovery, error) {
    proxy,inter := splitAcePath(servicePath)
    return NewAceDiscovery(proxy,inter,d.d)
}

func (d *AceDiscovery) SetFilter(filter ServiceDiscoveryFilter) {
    d.filter = filter
}

func (d *AceDiscovery) GetServices() []*KVPair {
    d.pairsMu.RLock()
    defer d.pairsMu.RUnlock()
    return d.pairs
}

func (d *AceDiscovery) WatchService() chan []*KVPair {
    d.mu.Lock()
    defer d.mu.Unlock()

    ch := make(chan []*KVPair, 10)
    d.chans = append(d.chans, ch)
    return ch
}

func (d *AceDiscovery) RemoveWatcher(ch chan []*KVPair) {
    d.mu.Lock()
    defer d.mu.Unlock()

    var chans []chan []*KVPair
    for _,c :=range d.chans {
        if c != ch {
            chans = append(chans, c)
        }
    }
    d.chans = chans
}

func (d *AceDiscovery) lookup() {
    var pairs []*KVPair
    err, srvs := GetAceCenterClient().getProxyInfos(d.proxy,d.inter)
    if err != nil {
        log.Warnf("get proxy infos error. %s:%s",d.proxy,d.inter)
        return
    }
    for _, info :=range srvs {
        pair := &KVPair{Key: fmt.Sprintf("ace@%s:%s",info.Host,info.Port),Value: fmt.Sprintf("%d",info.Status)}
        if d.filter != nil && !d.filter(pair) {
            continue
        }
        pairs = append(pairs, pair)
    }

    if len(pairs) > 0 {
        sort.Slice(pairs, func(i, j int) bool {
            return pairs[i].Key < pairs[j].Key
        })
    }

    d.pairsMu.Lock()
    d.pairs = pairs
    d.pairsMu.Unlock()

    d.mu.Lock()
    for _, ch := range d.chans {
        ch := ch
        go func() {
            defer func() {
                recover()
            }()
            select {
            case ch<- pairs:
            case <-time.After(time.Minute):
                log.Warn("chan is full and new change has been dropped")
            }
        }()
    }
    d.mu.Unlock()
}

func (d *AceDiscovery) watch() {
    tick := time.NewTicker(d.d)
    defer tick.Stop()

    for {
        select {
        case <-d.stopCh:
            return
        case <-tick.C:
            d.lookup()
        }
    }
}

func (d *AceDiscovery) Close() {
    close(d.stopCh)
}

