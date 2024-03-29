# DAFU(达夫) DNS Server

dafu 是一个专用的私有dns服务器，例如在基于IPv6的物联网中，我们想个每一个设备绑定一个私有的域名(sn(设备sn号).wida.cool)，这个时候如果在域名服务商中去绑定域名就不合适了。
我们可用的做法是在域名服务商中添加一条NS记录（将子域名解析到其他服务器）到我们的私有域名服务器中。

例如 
![](./doc/img/1.png)

然后我们自己的dns服务器实时动态的更新 域名到IPv6的地址。


通过http接口可以实时更新dns信息。设备可以每隔一端时间主动向服务端发送更新Ip地址请求。


## 使用

运行程序

```golang
$ go run main.go 
```

添加记录

```bash
$ curl -d "d=www.test.com&ip=98.23.22.22" http://localhost:9898/add
```

dig测试

```bash
$ dig a.test.com @127.0.0.1:8053
```