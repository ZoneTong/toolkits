// 测试将标准输入输出分身
package main

import (
	"io"
	"os"
	"sync"

	"github.com/zonetong/toolkits/errlog"
	"github.com/zonetong/toolkits/multiple"
)

// go run main.go <  in.txt > out.txt
// n倍输出
func main() {
	const n = 3

	// 1. sender
	done := make(chan bool)
	rs := multiple.CopiedReader(os.Stdin, n, done)

	// 2. receiver
	ws := multiple.MergedWriter(os.Stdout, n, done)

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
