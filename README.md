# leon-stream

![Build](https://github.com/sh2nk/leon-stream/actions/workflows/go.yml/badge.svg)

### Simple bot for mailing subscribers when streamer is online

Make sure that you exported all necessary environment variables, otherwise they'll fallback to dummy values.

Example `config.env`:

```dosini
VK_TOKEN="t0k3nex4mpl3"
POSTGRES_URL="postgres://user:password@localhost:5432/s3cr3t"
TWITCH_CLIENT_ID="fak3idixn8jqlgtr6n045c6plymhir"
TWITCH_SECRET="fak3s3cr37peo88hl2erzjggg0k30c"
BROADCASTER_ID=1337
DOMAIN="https://example.com"
APP_PORT=":8081"
```