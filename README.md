# toolkits

工具集， 包含按模板解析，及按模板生成

| 名称           | 作用       | 备注                                               |
| -------------- | ---------- | -------------------------------------------------- |
| http_server    | http服务   | 1. 接收任意http请求打印以调试;                     |
|                |            | 2. 将本地文件系统映射为/fs下的url                  |
|                |            | 3. 匀速下载                                        |
| echo_server    | 回声服务   | 接收tcp或udp数据, 立即返回相同或倒置或翻倍的字符串 |
| wr_client      | 写读客户端 | 配合翻倍服务,可测试实际读写tcp/udp的最大带宽       |
| uniform_client | 匀速客户端 | 匀速推送,可用echo_server配合接收                   |

## 使用示例

### http_server

1. http请求接收服务.打印http请求参数,cookies和实体
2. 系统文件服务.例如:http://127.0.0.1:12345/fs/tmp/1.txt 对应目录 /tmp/1.txt
3. 匀速下载服务.例如:http://127.0.0.1:12345/const/tmp/1.txt?cycle=true&speed=1 默认以1MBps速度传输给请求端

### echo_server

- 机器192.168.0.109命令:  echo_server/echo_server
- 然后任意机器测试命令:   echo_server/cmd.sh xnc 2 1 192.168.0.109 12345

### wr_client

- 服务器 182.242.45.69百倍返回: echo_server -m 100
- 客户端 (1.txt中是比较长的字符串) : cat /tmp/1.txt | wr_client --cycle -buf 3000

### uniform_client

- 服务器 echo_server -p 39999 -m 0
- 客户端 uniform_client -p 39999 -f /tmp/linux.tgz -speed=1 -c 1
