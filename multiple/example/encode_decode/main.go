// 测试将标准输入输出分身
package main

import (
	"io"
	"os"
	"sync"

	"github.com/ZoneTong/toolkits/errlog"
	"github.com/ZoneTong/toolkits/multiple"
)

// go run main.go <  in.txt > out.txt
// 多倍输出编码,多倍输入解码
func main() {
	const n = 3

	// 1. sender
	done := make(chan bool)
	var encoder MultipleEncoder
	rs := multiple.CopiedReader(os.Stdin, n, done, encoder.Encode)

	// 2. receiver
	var decoder MultipleDecoder
	ws := multiple.MergedWriter(os.Stdout, n, done, decoder.Decode)

	// 3. transfer
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			n, err := io.Copy(ws[i], rs[i])
			errlog.Log("Stdout copy over ", n, err)

		}(i)
	}

	wg.Wait()
	close(done)
}
