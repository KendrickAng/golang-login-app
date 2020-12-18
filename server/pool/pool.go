package pool

import (
	"encoding/gob"
	"fmt"
	"net"
)

type Pool interface {
	Get() (TcpConn, error)
	Put(*TcpConn)
	Destroy(*TcpConn) error
	Stats()
}

type TcpPool struct {
	config      TcpPoolConfig
	connections chan TcpConn
	total       uint
	alloced     uint
	reused      uint
}

type TcpPoolConfig struct {
	InitialSize int
	MaxSize     int
	Factory     func() (net.Conn, error)
}

type TcpConn struct {
	Enc  *gob.Encoder
	Dec  *gob.Decoder
	Conn net.Conn
}

func newTcpConn(f func() (net.Conn, error)) (TcpConn, error) {
	conn, err := f()
	if err != nil {
		return TcpConn{}, err
	}
	return TcpConn{
		Enc:  gob.NewEncoder(conn),
		Dec:  gob.NewDecoder(conn),
		Conn: conn,
	}, nil
}

func (pool *TcpPool) NewTcpPool(config TcpPoolConfig) Pool {
	pool.connections = make(chan TcpConn, config.MaxSize)
	pool.config = config
	for i := 0; i < pool.config.InitialSize; i++ {
		tcpConn, err := newTcpConn(pool.config.Factory)
		if err != nil {
			panic(err)
		}
		pool.connections <- tcpConn
	}
	return pool
}

func (pool *TcpPool) Get() (TcpConn, error) {
	pool.total++
	select {
	case conn := <-pool.connections:
		// return a pool connection if possible
		pool.reused++
		return conn, nil
	default:
		// dial a new conn if not enough in pool
		pool.alloced++
		tcpConn, err := newTcpConn(pool.config.Factory)
		return tcpConn, err
	}
}

func (pool *TcpPool) Put(tcpConn *TcpConn) {
	if tcpConn == nil {
		return
	}
	select {
	case pool.connections <- *tcpConn:
		return
	default:
		// if the pool is full, throw the connection away
		tcpConn.close()
	}
}

// Close the connection, don't put it back to pool
func (pool *TcpPool) Destroy(conn *TcpConn) error {
	err := conn.close()
	return err
}

func (pool *TcpPool) Stats() {
	fmt.Printf("Total: %v, Allocated: %v, Reused: %v\n", pool.total, pool.alloced, pool.reused)
}

func (conn *TcpConn) close() error {
	err := conn.Conn.Close()
	return err
}
