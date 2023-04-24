package client

import (
	"bufio"
	"errors"
	"fmt"
	//"io"
	"net"
	"time"

	"xace/log"
    //"xace/share"
)

type ConnFactoryFn func(c *Client, network, address string) (net.Conn, error)

var ConnFactories = map[string]ConnFactoryFn{
    "tcp": newDirectConn,
    "ace": newDirectConn,
    /*
	"http": newDirectHTTPConn,
	"kcp":  newDirectKCPConn,
	"quic": newDirectQuicConn,
	"unix": newDirectConn,
	"memu": newMemuConn,
    */
}

// Connect connects the server via specified network.
func (client *Client) Connect(network, address string) error {
	var conn net.Conn
	var err error

	switch network {
    /*
	case "http":
		conn, err = newDirectHTTPConn(client, network, address)
	case "ws", "wss":
		conn, err = newDirectWSConn(client, network, address)
	default:
		fn := ConnFactories[network]
		if fn != nil {
			conn, err = fn(client, network, address)
		} else {
			conn, err = newDirectConn(client, network, address)
		}
        */

    case "tcp","ace","":
		conn, err = newDirectConn(client, "tcp", address)
	default:
        log.Warnf("failed network: %s to dial server.", network)
        return errors.New("won't supoort network")
	}

	if err == nil && conn != nil {
		if tc, ok := conn.(*net.TCPConn); ok && client.option.TCPKeepAlivePeriod > 0 {
			_ = tc.SetKeepAlive(true)
			_ = tc.SetKeepAlivePeriod(client.option.TCPKeepAlivePeriod)
		}

		if client.option.IdleTimeout != 0 {
			_ = conn.SetDeadline(time.Now().Add(client.option.IdleTimeout))
		}

		if client.Plugins != nil {
			conn, err = client.Plugins.DoConnCreated(conn)
			if err != nil {
				return err
			}
		}

		client.Conn = conn
		client.r = bufio.NewReaderSize(conn, ReaderBuffsize)
		// c.w = bufio.NewWriterSize(conn, WriterBuffsize)

		// start reading and writing since connected
		go client.input()

		if client.option.Heartbeat && client.option.HeartbeatInterval > 0 {
			go client.heartbeat()
		}

	}

	if err != nil && client.Plugins != nil {
		client.Plugins.DoConnCreateFailed(network, address)
	}

	return err
}

func newDirectConn(c *Client, network, address string) (net.Conn, error) {
	var conn net.Conn
	var err error

	if c == nil {
		err = fmt.Errorf("nil client")
		return nil, err
	}
	conn, err = net.DialTimeout(network, address, c.option.ConnectTimeout)

	if err != nil {
		log.Warnf("failed to dial server: %v", err)
		return nil, err
	}
	return conn, nil
}


