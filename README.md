A http proxy which will implement multi-process transport but respond sequentially

多线程且按先后顺序获取数据，能够在提高获取速度的同时防止下载大文件断流，适合在线观看视频（或“边下边放”）

### Status

[![Travis Build Status](https://travis-ci.org/mozikachu/antorange.svg?branch=master)](https://travis-ci.org/mozikachu/antorange)

### Get Started

1. `go get github.com/mozikachu/antorange` or download the binary file from [releases](https://github.com/mozikachu/antorange/releases)
2. Execute `antorange`, it will listen on port 1234 as default.
3. Make `127.0.0.1:1234` as the proxy server of your client (such as browser, wget...)

### Advanced Usage

```
-> $ antorange --help
Usage of antorange:
  -b int
        <bufferSize> per range (default 2097152) // 每次 "range" 获取的数据大小，默认是 2 MiB
  -l string
        addr to listen (default ":1234") // 代理服务器监听的端口，一般情况下是 127.0.0.1:1234
  -r int
        retry times (default 3) // 失败重试次数，默认为 3
  -t int
        concurrent range (default 3) // 同时进行的最大传输数，默认是 3
```

### TODO


- [ ] HTTPS multi-process transport support
- [ ] Upload multi-process transport support

### Snapshot

![antorange is working](https://i.imgur.com/iBkd09D.jpg)