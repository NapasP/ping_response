### Config
build/conf.toml

### Installation
**Windows**

Install in your server and start file build/respone.exe.

**Linux**
```sh
$ git clone https://github.com/NapasP/ping_response.git
$ cd build
$ chmod 775 respone
$ screen -d -m ./respone
```

### Create bot telegram
1. Add to contact botfather in telegram;
2. Press start;
3. We write /newbot;
4. Set name;
5. Copy token key and insert to config.

#### How to find chat id telegram
1. Add bot to new chat;
2. Write any message in chat;
3. Go to the URL https://api.telegram.org/botTOKEN_BOT/getUpdates;
4. Copy "id"
