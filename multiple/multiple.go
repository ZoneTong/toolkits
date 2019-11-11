package multiple

import (
	"io"
	"sync"

	"github.com/ZoneTong/toolkits/errlog"
)

const (
	MTU = 1500
)

var (
	pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, MTU)
		},
	}
)

// 将一个reader共享给n个分身reader, 同一份输入复制成n分输入
func CopiedReader(reader io.Reader, multiple uint32, done <-chan bool, filters ...Filter) (inputs []io.Reader) {
	var uplinkWriters []*io.PipeWriter
	for i := uint32(0); i < multiple; i++ {
		r, w := io.Pipe()
		inputs = append(inputs, r)
		uplinkWriters = append(uplinkWriters, w)
	}

	// input
	go func() {
		var err error
		var n int
		defer func() {
			for _, w := range uplinkWriters {
				w.Close()
			}
			<-done
			errlog.Log("CopiedReader ", n, err)
		}()

		for {
			var m = pool.Get().([]byte)
			n, err = reader.Read(m)
			if err != nil { // && errors.Cause(err) != io.EOF
				return
			}
			m = m[:n]

			// TODO: format data
			for _, handle := range filters {
				m = handle(m)
			}
			bs := pool.Get().([]byte)[:len(m)]
			copy(bs, m)
			pool.Put(m)

			for _, sender := range uplinkWriters {
				sender.Write(bs)
			}
			pool.Put(bs)
		}
	}()
	return
}

// 将一个writer分身成n个, 并将接收的输出合并成一个
func MergedWriter(writer io.Writer, n uint32, done <-chan bool, filters ...Filter) (outputs []io.Writer) {
	var dwlinkReaders []*io.PipeReader
	for i := uint32(0); i < n; i++ {
		r, w := io.Pipe()
		outputs = append(outputs, w)
		dwlinkReaders = append(dwlinkReaders, r)
	}

	ch := make(chan []byte)
	recvfunc := func(recv io.Reader) {
		var err error
		for {
			var n int
			var m = pool.Get().([]byte)
			n, err = recv.Read(m)
			if err != nil {
				return
			}

			ch <- m[:n]
		}
	}

	for _, r := range dwlinkReaders {
		go recvfunc(r)
	}

	go func() {
		var err error
		defer func() {
			for _, r := range dwlinkReaders {
				r.Close()
			}
			errlog.Log("MergedWriter finish ", err)
		}()
		for {
			select {
			case <-done:
				return

			case m := <-ch:

				// TODO: check data format integrity
				for _, handle := range filters {
					m = handle(m)
				}

				if len(m) == 0 {
					continue
				}

				_, err = writer.Write(m)
				pool.Put(m)
				if err != nil {
					return
				}
			}
		}
	}()
	return
}

type Filter func([]byte) []byte
