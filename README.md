# Build
Tusk: the sharpest Go starter for building APIs
## Usage

Create `.env` from template 

```bash
cp .env.example .env
```

Populate `.env` with appropriate values and run the server

```bash
go run cmd/main.go
```



## 🤔Why does this exist?

I build a lot of random projects—most of them never make it past localhost 😅. Eventually, I felt the need to pull out the commonly used pieces and turn them into a structured starting point.

This setup probably won’t work for everyone—or maybe even most people writing Go—but I’m putting it out there anyway.
It’s mainly for my own sanity... and future me.

## TODO / Improvements
 Add CORS & Logging middleware

 Add Docker support

 Write tests for handlers and repo

 Switch to chi or fiber later

 Add Swagger/OpenAPI docs

bye</br>

