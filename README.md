# toolkits

工具集， 包含按模板解析，及按模板生成

| 名称        | 作用     | 备注                                                              |
| ----------- | -------- | ----------------------------------------------------------------- |
| http_server | http服务 | 1. 接收任意http请求打印以调试;                                    |
|             |          | 2. 将本地文件系统映射为/fs下的url                                 |
|             |          | 3. 匀速下载                                                       |
| echo_server | 回声服务 | 接收tcp或udp数据, 立即返回相同或倒置的字符串,可配合nc命令调试使用 |
| rw_server   | 读写服务 | 翻数倍返回客户端请求数据,以测试实际读写tcp/udp的最大带宽          |

## 使用示例

### http_server

1. http请求接收服务.打印http请求参数,cookies和实体
1. 系统文件服务.例如:http://127.0.0.1:12345/fs/tmp/1.txt 对应目录 /tmp/1.txt
1. 匀速下载服务.例如:http://127.0.0.1:12345/const/tmp/1.txt?cycle=true&speed=1 默认以1MBps速度传输给请求端

### echo_server

机器192.168.0.109命令:  echo_server/echo_server
然后任意机器测试命令:   echo_server/cmd.sh xnc 2 1 192.168.0.109 12345

### rw_server

服务器 182.242.45.69百倍返回: rw_server -u -m 100 -z
客户端 (1.txt中是比较长的字符串) : cat /tmp/1.txt | rw_client -u -h 182.242.45.69 -z
