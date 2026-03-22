# 🐱 RoboKaty

> A powerful, Rose-style Telegram group management bot — built in **Golang** with gotgbot v2.

---

## ✨ Features

| Module | Commands |
|---|---|
| 🛡️ Admin | ban, tban, kick, warn, mute, promote, purge, pin, report... |
| 📝 Notes | save, #notename, delnote, deleteall |
| 🔍 Filters | filter, filters, stop, stopall |
| 👋 Welcome | toggle_welcome, setwelcome, resetwelcome |
| 🔒 Locks | lock, unlock, locks |
| 📋 Rules | rules, setrules, resetrules |
| 🚫 Blacklist | blacklist, blacklisted, whitelist |
| 😴 AFK | afk |
| ⭐ Karma | karma, karma_toggle |
| 🌙 Nightmode | nightmode -s=22:00 -e=6h |
| 🏓 Ping | ping |
| 🔤 Sed | s/old/new/flags |
| 📡 Broadcast | broadcast (sudo only) |
| 🎭 Stickers | getsticker, kang |
| 🎌 Anime | anime, manga, character |
| 📖 Urban | ud, urban |
| 💬 Quotly | q (quote sticker) |
| 🔎 Sangmata | sangmata (name history) |
| ✅ AutoApprove | approve, disapprove, approved |
| 👢 InKick | inkick, softban, ban_ghosts |
| 📦 Dev | stats, gban, shell, logs, restart |

---

## ⚡ Command Prefixes

All commands work with **any** of these prefixes:
```
/start    ← standard
.start    ← dot prefix  
!start    ← exclamation prefix
```

---

## 🚀 Deployment

### 1. Clone & Setup

```bash
git clone https://github.com/yourname/robokaty
cd robokaty
cp config.env.sample config.env
nano config.env   # fill in your values
```

### 2. Install Go (Ubuntu/AWS)

```bash
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### 3. Setup PostgreSQL

```bash
sudo apt install postgresql -y
sudo -u postgres psql
CREATE USER robokaty WITH PASSWORD 'yourpassword';
CREATE DATABASE robokatydb OWNER robokaty;
\q
```

### 4. Build & Run

```bash
go mod tidy
go build -o robokaty .
./robokaty
```

### 5. Run as Service (systemd)

```bash
sudo nano /etc/systemd/system/robokaty.service
```

```ini
[Unit]
Description=RoboKaty Telegram Bot
After=network.target postgresql.service

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/robokaty
ExecStart=/home/ubuntu/robokaty/robokaty
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable robokaty
sudo systemctl start robokaty
sudo systemctl status robokaty
```

### 6. Docker (Alternative)

```bash
docker build -t robokaty .
docker run -d --env-file config.env --name robokaty robokaty
```

---

## 📞 Support

Channel: [@RoboKaty](https://t.me/RoboKaty)

---

## 📄 License

MIT License — feel free to use and modify.
