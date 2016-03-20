A http proxy which will implement multi-process transport but respond sequentially

多线程且按先后顺序获取数据，能够在提高获取速度的同时防止下载大文件断流，适合在线观看视频（或“边下边放”）

### Status

[![Travis Build Status](https://travis-ci.org/mozikachu/antorange.svg?branch=master)](https://travis-ci.org/mozikachu/antorange)

### Usage

1. `go get github.com/mozikachu/antorange` or download the binary file from [releases](https://github.com/mozikachu/antorange/releases)
2. Launch `antorange`, it will listen on port 1234 as default.
   The following command `./antorange --help` will show you more information
3. Make `http://127.0.0.1:1234` as the proxy server of your client (such as browser, wget...)

### TODO


- [ ] HTTPS multi-process transport support
- [ ] Upload multi-process transport support


