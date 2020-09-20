# Supported Messages

This is about the messages type received from the Message Queue system

## Type: SQL

Example

```json
{
    "TYPE":"SQL",
    "CONTENT":{
        "SERVER":"localhost",
         "DB":"postgresql",
         "USER":"postgres",
         "PASS":"mysecretpassword",
         "SENTENCE":"SELECT pg_sleep(1.5);"
    },
    "DATE":"2020-01-01 00:00:01.000000-1",
    "APPID":"test",
    "ADITIONAL":null,
    "ACK": false,
    "RESPONSE":null
}
```
