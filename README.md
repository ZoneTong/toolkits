# toolkits

工具集， 包含按模板解析，及按模板生成

| 名称        | 作用         | 备注                                                               |
| ----------- | ------------ | ------------------------------------------------------------------ |
| http_server | 静态http服务 | 1. 接收任意http调试请求; 2.也可将本地文件系统映射为/fs下的远程目录 |
| echo_server | 回声服务     | 接收tcp或udp数据, 立即返回倒置的字符串或相同字符串,配合nc命令使用  |
| rw_server   | 读写服务     | 测试客户端服务器之间实际读写tcp/udp的带宽,配合nload测试                                     |

## 使用示例

### http_server

http://127.0.0.1:12345/fs/tmp/1.txt 对应目录 /tmp/1.txt

### echo_server

机器192.168.0.109命令:  echo_server/echo_server
然后任意机器测试命令:   echo_server/cmd.sh xnc 2 1 192.168.0.109 12345

### rw_server

服务器 182.242.45.69: rw_server -u -m 100 -z
客户端 (1.txt中是比较长的字符串) : cat /tmp/1.txt | rw_client -u -h 182.242.45.69 -z
