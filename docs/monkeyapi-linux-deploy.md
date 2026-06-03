# MonkeyAPI Linux 服务器部署与更新说明

本文档适用于仓库：

```text
https://github.com/monkey-Akira/monkeyapi.git
```

当前推荐部署方式：

1. 代码推送到 GitHub。
2. GitHub Actions 自动构建 Docker 镜像。
3. 服务器只拉取镜像并启动容器。

服务器不要再执行 `--build`。`--build` 会让服务器本地编译前端和后端，4G 内存机器容易卡死或被系统 kill。

## 1. GitHub 镜像构建

仓库已经包含 workflow：

```text
.github/workflows/docker-image-main.yml
```

每次推送 `main` 分支后，GitHub 会自动构建镜像：

```text
ghcr.io/monkey-akira/monkeyapi:latest
```

你可以在 GitHub 仓库页面查看：

```text
Actions -> Publish Docker image (main)
```

只有 Actions 变成绿色成功后，服务器才能拉到最新镜像。

如果 GHCR package 不是公开的，服务器需要先登录一次：

```bash
docker login ghcr.io
```

用户名填 GitHub 用户名，密码填 GitHub Personal Access Token。更简单的方式是把 package 设置成 Public。

## 2. 首次拉取代码

建议放在 `/monkey/opt/monkeyapi`：

```bash
mkdir -p /monkey/opt
cd /monkey/opt
git clone https://github.com/monkey-Akira/monkeyapi.git
cd /monkey/opt/monkeyapi
```

确认当前代码：

```bash
git branch
git log --oneline -5
```

## 3. 配置环境变量

只需要使用 `.env.production`，不要改 `.env.example`。

```bash
cd /monkey/opt/monkeyapi
cp .env.production.example .env.production
nano .env.production
```

没有域名、先通过宝塔或 Nginx 反代时，可以这样写：

```env
FRONTEND_BASE_URL=http://你的服务器IP:3000
HOST_BIND=127.0.0.1
APP_PORT=3000
TZ=Asia/Shanghai
NODE_NAME=monkeyapi-node-1

POSTGRES_USER=root
POSTGRES_PASSWORD=MonkeyApi_DB_123456
POSTGRES_DB=new-api

REDIS_PASSWORD=MonkeyApi_Redis_123456
SESSION_SECRET=MonkeyApi_Session_1234567890_abcdef
```

说明：

- `HOST_BIND=127.0.0.1` 表示只允许服务器本机访问 3000 端口，推荐配合宝塔/Nginx 反代。
- 如果你想临时直接访问 `服务器IP:3000`，改成 `HOST_BIND=0.0.0.0`，同时安全组/防火墙要放行 3000。
- `POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`SESSION_SECRET` 不能留空。
- 密码建议只用英文、数字、下划线、短横线和点号。
- `.env.production` 不要提交到 GitHub。

有域名和 HTTPS 后，再把：

```env
FRONTEND_BASE_URL=https://你的域名
```

## 4. 首次启动

先拉镜像，再启动：

```bash
cd /monkey/opt/monkeyapi
docker compose --env-file .env.production -f docker-compose.prod.yml pull
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

查看状态：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker logs --tail=100 monkeyapi
```

正常状态类似：

```text
monkeyapi            Up ... 127.0.0.1:3000->3000/tcp
monkeyapi-postgres   Up ... healthy
monkeyapi-redis      Up ... healthy
```

日志里看到下面内容说明应用已启动：

```text
New API ... started
ready in ...
```

本机测试：

```bash
curl http://127.0.0.1:3000/api/status
```

返回里有 `"success":true` 就正常。

## 5. 宝塔或 Nginx 反代

如果 `.env.production` 使用：

```env
HOST_BIND=127.0.0.1
APP_PORT=3000
```

宝塔反代目标填写：

```text
http://127.0.0.1:3000
```

Nginx 示例：

```nginx
server {
    listen 80;
    server_name 你的域名;

    client_max_body_size 100m;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_buffering off;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
    }
}
```

宝塔配置 HTTPS 后，把 `.env.production` 里的 `FRONTEND_BASE_URL` 改成 HTTPS 域名，然后重启：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

## 6. 日常更新

每次代码推送到 GitHub 后，先等 Actions 构建成功。

服务器执行：

```bash
cd /monkey/opt/monkeyapi
git pull
docker compose --env-file .env.production -f docker-compose.prod.yml pull
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

不要执行：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

## 7. 常用命令

查看容器：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

查看应用日志：

```bash
docker logs -f monkeyapi
```

重启应用：

```bash
docker restart monkeyapi
```

停止整套服务：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml down
```

不要加 `-v`，否则会删除数据库卷。

## 8. 常见问题

### 3000 端口被占用

报错：

```text
Bind for 0.0.0.0:3000 failed: port is already allocated
```

查看占用：

```bash
ss -lntp | grep :3000
docker ps --format "table {{.ID}}\t{{.Names}}\t{{.Ports}}\t{{.Image}}"
```

如果是旧容器占用，停掉旧容器：

```bash
docker stop 旧容器名
docker rm 旧容器名
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

如果 3000 必须给别的服务使用，改 `.env.production`：

```env
APP_PORT=3001
```

然后重启：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

宝塔反代目标也要改成：

```text
http://127.0.0.1:3001
```

### 应用连不上 postgres

报错：

```text
lookup postgres on 127.0.0.53:53: read: connection refused
```

先检查网络：

```bash
docker inspect monkeyapi --format '{{json .NetworkSettings.Networks}}'
```

如果输出是 `{}`，说明应用容器没有加入 compose 网络。修复：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml stop new-api
docker rm monkeyapi
docker compose --env-file .env.production -f docker-compose.prod.yml up -d new-api
```

再检查：

```bash
docker inspect monkeyapi --format '{{json .NetworkSettings.Networks}}'
docker logs --tail=80 monkeyapi
```

正常应该能看到：

```text
monkeyapi_monkeyapi-network
```

### record not found

首次启动时日志里出现：

```text
record not found
system is not initialized and no root user exists
```

这是正常的，表示新数据库还没有初始化管理员账号。打开前台按初始化流程创建管理员即可。

## 9. 备份

更新前建议备份数据库卷和本地数据：

```bash
cd /monkey/opt/monkeyapi
mkdir -p /monkey/opt/monkeyapi-backup
docker run --rm -v monkeyapi_pg_data:/volume -v /monkey/opt/monkeyapi-backup:/backup alpine \
  tar czf /backup/pg_data_$(date +%F_%H%M%S).tar.gz -C /volume .
tar czf /monkey/opt/monkeyapi-backup/data_$(date +%F_%H%M%S).tar.gz data logs
```
