# Go Offensive Fortune

A lightweight Go reimplementation of the offensive fortune web app.

## Features

- Reads fortune databases from `/usr/share/games/fortunes/off/`
- Supports `.dat` + content files using the Unix fortune binary format
- Falls back to plaintext `%`-delimited fortunes if needed
- Rotates ROT13 text when the fortune flags require it
- Serves a minimal HTML page with a black-on-white-and-monospaced style
- Logs each root request to a SQLite database in `data/requests.db`

## Run

```bash
cd /home/cdonges/src/go-offensive-fortune
go run .
```

Then open:

- http://localhost:6724/

## Options

```bash
go run . -dir /usr/share/games/fortunes/off/ -addr :6724 -db data/requests.db
```
