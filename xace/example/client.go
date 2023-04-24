package main

import (
    "fmt"
    "time"
    "context"

    "xace/client"
    "xace/protocol"
    //"xace/codec"
)

func runxclient() {
    dc,_ := client.NewPeer2PeerDiscovery("10.0.10.39:8188","")

    client := client.NewXClient("PublicConf", client.Failover, client.RandomSelect, dc, client.AceOption)

    ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
    var dest int64 = 13
    var desc string = "tst desc"
    var args []any = make([]any,0,2)
    args = append(args,dest)
    args = append(args,desc)
    var reply []any = make([]any,0, 2)
    var str string
    reply = append(reply, &str)
    err := client.Call(ctx, "add", args, reply)
    if err != nil {
        fmt.Println("call error. ", err)
    } else {
        fmt.Println(reply,str)
    }

    time.Sleep(100*time.Second)
    cancel()
}

func runclient() {
    client := client.NewClient(client.DefaultOption)//new(client.Client)
    err := client.Connect("tcp","10.0.10.39:8188")
    if err != nil {
        fmt.Println("err ",err)
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
    var dest int64 = 13
    var desc string = "tst desc"
    //args := codec.EncodeArgs(&dest, &desc)
    var args []any = make([]any,0,2)
    args = append(args,dest)
    args = append(args,desc)
    var reply []any = make([]any,0, 2)
    var str string
    reply = append(reply, &str)
    err = client.Call(ctx, "PublicConf","add", args, reply)
    if err != nil {
        fmt.Println("call error. ", err)
    } else {
        fmt.Println(reply,str)
    }

    time.Sleep(100*time.Second)
    cancel()
}

func runx() {
    aclient := client.GetAceClient()

    ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
    var dest int64 = 13
    var desc string = "tst desc"
    var args []any = make([]any,0,2)
    args = append(args,dest)
    args = append(args,desc)
    var str string

    reply := protocol.GetAceReply()
    reply.Args = append(reply.Args, &str)
    //var reply []any = make([]any,0, 2)
    //reply = append(reply, &str)

    err := aclient.Call(ctx, "PublicConf.PublicConf", "add", args, reply)
    if err != nil {
        fmt.Println("call error. ", err)
    } else {
        fmt.Println(reply.Retcode, str)
    }

    cancel()
}

func main() {
    client.InitializeAceCenter("10.0.10.103:16999", "AaceCenter")
    runx()
    time.Sleep(1*time.Second)
    runx()
    time.Sleep(1*time.Second)
    runx()
    time.Sleep(100*time.Second)
}

