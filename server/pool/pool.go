package pool

import (
	"fmt"
	"log"
	"net"
)

type Pool interface {
	Get() (net.Conn, error)
	Put(net.Conn)
	Destroy(net.Conn) error
	Stats()
}

type TcpPool struct {
	config      TcpPoolConfig
	connections chan net.Conn
	total       uint
	alloced     uint
	reused      uint
}

type TcpPoolConfig struct {
	InitialSize int
	MaxSize     int
	Factory     func() (net.Conn, error)
}

func (pool *TcpPool) NewTcpPool(config TcpPoolConfig) Pool {
	pool.connections = make(chan net.Conn, config.MaxSize)
	pool.config = config
	for i := 0; i < pool.config.InitialSize; i++ {
		conn, err := pool.config.Factory()
		if err != nil {
			log.Panicln(err)
		}
		pool.connections <- conn
	}
	return pool
}

func (pool *TcpPool) Get() (net.Conn, error) {
	pool.total++
	select {
	case conn := <-pool.connections:
		// return a pool connection if possible
		pool.reused++
		return conn, nil
	default:
		// dial a new conn if not enough in pool
		pool.alloced++
		conn, err := pool.config.Factory()
		return conn, err
	}
}

func (pool *TcpPool) Put(conn net.Conn) {
	if conn == nil {
		return
	}
	select {
	case pool.connections <- conn:
		return
	default:
		// if the pool is full, throw the connection away
		conn.Close()
	}
}

// Close the connection, don't put it back to pool
func (pool *TcpPool) Destroy(conn net.Conn) error {
	err := conn.Close()
	return err
}

func (pool *TcpPool) Stats() {
	fmt.Printf("Total: %v, Allocated: %v, Reused: %v\n", pool.total, pool.alloced, pool.reused)
}
