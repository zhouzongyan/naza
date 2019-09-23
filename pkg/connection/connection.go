// package connection
//
// 对 net.Conn 接口的二次封装，目的有两个：
// 1. 在流媒体传输这种特定的长连接场景下提供更方便、高性能的接口
// 2. 便于后续将 TCPConn 替换成其他传输协议
package connection

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

var connectionErr = errors.New("connection: fxxk")

type Connection interface {
	// 包含 interface net.Conn 的所有方法
	net.Conn

	ReadAtLeast(buf []byte, min int) (n int, err error)
	ReadLine() (line []byte, isPrefix bool, err error)

	Printf(fmt string, v ...interface{}) (n int, err error)

	ModWriteBufSize(n int)
	ModReadTimeoutMS(n int)
	ModWriteTimeoutMS(n int)
}

type Config struct {
	// 如果不为0，则之后每次读/写使用 buffer 缓冲
	ReadBufSize  int
	WriteBufSize int

	// 如果不为0，则之后每次读/写都带超时
	ReadTimeoutMS  int
	WriteTimeoutMS int
}

func New(conn net.Conn, config Config) Connection {
	var c connection
	c.Conn = conn
	if config.ReadBufSize > 0 {
		c.r = bufio.NewReaderSize(conn, config.ReadBufSize)
	} else {
		c.r = conn
	}
	if config.WriteBufSize > 0 {
		c.w = bufio.NewWriterSize(conn, config.WriteBufSize)
	} else {
		c.w = conn
	}
	c.config = config
	return &c
}

type connection struct {
	Conn   net.Conn
	r      io.Reader
	w      io.Writer
	config Config
}

// Mod类型函数不加锁

// 由调用方保证不和写操作并发执行
func (c *connection) ModWriteBufSize(n int) {
	if c.config.WriteBufSize > 0 {
		// 如果之前已经设置过写缓冲，直接 panic
		// 这里改成 flush 后替换成新缓冲也行，暂时没这个必要
		panic(connectionErr)
	}
	c.config.WriteBufSize = n
	c.w = bufio.NewWriterSize(c.Conn, n)
}

func (c *connection) ModReadTimeoutMS(n int) {
	if c.config.ReadTimeoutMS > 0 {
		panic(connectionErr)
	}
	c.config.ReadTimeoutMS = n
}

func (c *connection) ModWriteTimeoutMS(n int) {
	if c.config.WriteTimeoutMS > 0 {
		panic(connectionErr)
	}
	c.config.WriteTimeoutMS = n
}

func (c *connection) ReadAtLeast(buf []byte, min int) (n int, err error) {
	if c.config.ReadTimeoutMS > 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(time.Duration(c.config.ReadTimeoutMS) * time.Millisecond))
	}
	return io.ReadAtLeast(c.r, buf, min)
}

func (c *connection) ReadLine() (line []byte, isPrefix bool, err error) {
	bufioReader, ok := c.r.(*bufio.Reader)
	if !ok {
		// 目前只有使用了 bufio.Reader 时才能执行 ReadLine 操作
		panic(connectionErr)
	}
	if c.config.ReadTimeoutMS > 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(time.Duration(c.config.ReadTimeoutMS) * time.Millisecond))
	}
	return bufioReader.ReadLine()
}

func (c *connection) Printf(format string, v ...interface{}) (n int, err error) {
	if c.config.WriteTimeoutMS > 0 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(time.Duration(c.config.WriteTimeoutMS) * time.Millisecond))
	}
	return fmt.Fprintf(c.Conn, format, v...)
}

func (c *connection) Read(b []byte) (n int, err error) {
	if c.config.ReadTimeoutMS > 0 {
		_ = c.Conn.SetReadDeadline(time.Now().Add(time.Duration(c.config.ReadTimeoutMS) * time.Millisecond))
	}
	return c.r.Read(b)
}

func (c *connection) Write(b []byte) (n int, err error) {
	if c.config.WriteTimeoutMS > 0 {
		_ = c.Conn.SetWriteDeadline(time.Now().Add(time.Duration(c.config.WriteTimeoutMS) * time.Millisecond))
	}
	return c.w.Write(b)
}

func (c *connection) Close() error {
	w, ok := c.w.(*bufio.Writer)
	if ok {
		w.Flush()
	}
	return c.Conn.Close()
}

func (c *connection) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

func (c *connection) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

func (c *connection) SetDeadline(t time.Time) error {
	return c.Conn.SetDeadline(t)
}

func (c *connection) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}
func (c *connection) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}
