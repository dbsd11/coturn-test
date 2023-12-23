# 打镜像
docker build . -t answer-test:1.0

＃ 启动
docker run -ti --rm -p 60000:60000 answer-test:1.0
