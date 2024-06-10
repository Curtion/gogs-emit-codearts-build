# 监听Gogs Webhook并触发华为云CodeArtsBuild自动部署

修改`config.toml.default`为 `config.toml`，并修改其中配置， 保持`config.toml`处于可执行文件相同目录 

设置Gogs Webhook，URL为`http://yourdomain:port/webhook`

程序未校验密钥, 请自动保证Webhook安全性