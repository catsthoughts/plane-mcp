# Debug Build Guide

## Enabling Debug Mode

Edit `main.go` and set:

```go
const DEBUG = true
```

## What Debug Does

When `DEBUG = true`, all operations are logged to `/tmp/plane-mcp-debug.log`

## Log File Location

```
/tmp/plane-mcp-debug.log
```

The log is opened in append mode with immediate sync to ensure all writes are flushed.

## Disable Debug

Set `const DEBUG = false` and rebuild:

```bash
go build -o plane-mcp .
```