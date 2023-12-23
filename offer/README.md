# 打镜像
docker build . -t offer-test:1.0

# 启动，需知道信令服务公网地址，本demo直接用answer公网服务地址，但是一般不是一个
docker run -ti --rm -p 50000:50000  offer-test:1.0 -answer-address ${signal_server_public_address}:60000
