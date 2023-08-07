### 直接输入合并后目标文件

```shell
./merge -f xxx -o target.yml
```

### 合并后作为http服务（用于clash客户端订阅地址）

```shell
./merge -f xxx -p 8080
```

客户端可通过`http://localhost:8080`订阅

### 使用subConverter后端解析订阅地址

```shell
./merge -u xxx -u xxx -s http://127.0.0.1:25500 -p 8080
```