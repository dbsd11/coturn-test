docker run -d \
  -p 58080:8080 \
  -p 58081:58081 \
  -e NEKO_SCREEN=1024×576@30    \
  -e NEKO_PASSWORD=jQNMEDtcJR \
  -e NEKO_PASSWORD_ADMIN=juji_user \
  -e NEKO_NAT1TO1=1.95.59.148 \
  -e NEKO_TCPMUX=58081 \
  -e NEKO_UDPMUX=58081 \
  -e NEKO_ICELITE=1 \
  -e NEKO_VIDEO_CODEC=h264 \
  -e NEKO_ICESERVERS='[{ "urls": [ "turn:192.168.0.20:63478" ], "username":"user-test", "credential":"123456" }]' \
  --shm-size=512mb \
  --cap-add=SYS_ADMIN \
  --restart=unless-stopped \
  --name neko \
  swr.cn-southwest-2.myhuaweicloud.com/juji-develop/neko/chromium:latest
