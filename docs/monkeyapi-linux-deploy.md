# MonkeyAPI Linux 服务器部署与更新文档

本文档适用于把 `https://github.com/monkey-Akira/monkeyapi.git` 部署到 Linux 服务器，并在之后从 GitHub 拉取更新。

项目已经自带生产用 `docker-compose.prod.yml` 和 `.env.production.example`，不需要再手动编写 Docker Compose 文件。

## 1. 服务器准备

推荐系统：Ubuntu 22.04/24.04 或 Debian 12。

开放端口：

- `22`：SSH
- `80`：HTTP
- `443`：HTTPS

登录服务器：

```bash
ssh root@你的服务器IP
```

安装基础工具：

```bash
apt update
apt upgrade -y
apt install -y git curl wget vim nano ca-certificates gnupg lsb-release ufw openssl
```

配置防火墙：

```bash
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
ufw status
```

## 2. 安装 Docker 和 Docker Compose

Ubuntu/Debian 推荐使用 Docker 官方源：

```bash
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  > /etc/apt/sources.list.d/docker.list

apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

如果是 Debian，把上面命令中的 `https://download.docker.com/linux/ubuntu` 改成：

```text
https://download.docker.com/linux/debian
```

验证：

```bash
docker --version
docker compose version
systemctl enable docker
systemctl status docker
```

## 3. 拉取项目代码

建议放在 `/opt/monkeyapi`：

```bash
mkdir -p /opt
cd /opt
git clone https://github.com/monkey-Akira/monkeyapi.git
cd /opt/monkeyapi
```

确认分支和版本：

```bash
git branch
git log --oneline -5
```

## 4. 配置生产环境变量

项目已经自带 `docker-compose.prod.yml`，只需要复制环境变量模板：

```bash
cd /opt/monkeyapi
cp .env.production.example .env.production
nano .env.production
```

至少修改这些值：

```env
FRONTEND_BASE_URL=https://你的域名
POSTGRES_PASSWORD=改成强密码
REDIS_PASSWORD=改成强密码
SESSION_SECRET=改成随机字符串
```

生成 `SESSION_SECRET`：

```bash
openssl rand -hex 32
```

密码建议只使用英文字母、数字、下划线、短横线和点号。因为数据库连接串会使用 URL 格式，特殊符号需要 URL 编码，容易写错。

真实的 `.env.production` 已加入 `.gitignore`，不要提交到 GitHub。

## 5. 首次启动

```bash
cd /opt/monkeyapi
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

查看容器：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

查看日志：

```bash
docker logs -f monkeyapi
```

本机测试：

```bash
curl http://127.0.0.1:3000/api/status
```

如果返回内容里有 `"success":true`，说明服务已启动。

## 6. 配置 Nginx 反向代理

安装 Nginx：

```bash
apt install -y nginx
systemctl enable nginx
```

创建站点配置：

```bash
nano /etc/nginx/sites-available/monkeyapi.conf
```

写入以下内容，把 `你的域名` 改成真实域名：

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

启用站点：

```bash
ln -s /etc/nginx/sites-available/monkeyapi.conf /etc/nginx/sites-enabled/monkeyapi.conf
nginx -t
systemctl reload nginx
```

浏览器访问：

```text
http://你的域名
```

## 7. 配置 HTTPS

安装 Certbot：

```bash
apt install -y certbot python3-certbot-nginx
```

申请证书：

```bash
certbot --nginx -d 你的域名
```

测试自动续期：

```bash
certbot renew --dry-run
```

HTTPS 完成后，把 `/opt/monkeyapi/.env.production` 里的 `FRONTEND_BASE_URL` 改成：

```env
FRONTEND_BASE_URL=https://你的域名
```

然后重启应用：

```bash
cd /opt/monkeyapi
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

## 8. 日常更新

每次你在本地修改并推送到 GitHub 后，服务器执行：

```bash
cd /opt/monkeyapi
git pull
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

确认状态：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml ps
docker logs --tail=100 monkeyapi
```

## 9. 备份

建议更新前备份数据库卷和本地数据目录：

```bash
cd /opt/monkeyapi
mkdir -p /opt/monkeyapi-backup
docker run --rm -v monkeyapi_pg_data:/volume -v /opt/monkeyapi-backup:/backup alpine \
  tar czf /backup/pg_data_$(date +%F_%H%M%S).tar.gz -C /volume .
tar czf /opt/monkeyapi-backup/data_$(date +%F_%H%M%S).tar.gz data logs
```

## 10. 常用命令

启动或更新：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

停止：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml down
```

查看日志：

```bash
docker logs -f monkeyapi
```

重启：

```bash
docker restart monkeyapi
```

查看占用端口：

```bash
ss -lntp | grep 3000
```

## 11. 重要说明

- 生产部署使用 `docker-compose.prod.yml`，不要再手动写 Compose 文件。
- `.env.production` 保存真实密码，只放在服务器上，不提交到 GitHub。
- 项目默认把应用只绑定到 `127.0.0.1:3000`，公网通过 Nginx 访问。
- 更新代码后需要重新 `up -d --build`，否则 Docker 镜像不会包含最新代码。
- 如果修改了数据库密码或 Redis 密码，已有容器和数据卷可能需要同步处理，生产环境不要随意改。
