# hal
Read from a redis list and post to Slack. You know, all that chatops stuff.

## Usage
```
./hal       --redis_connection  string      default="127.0.0.1:6379"
            --redis_db          int         default=0
            --redis_list        string      default="mylist"
            --slack_channel     string      default=NONE (REQUIRED)
            --slack_token       string      default=NONE (REQUIRED)
            --watch_interval    int         default=3
            --debug             bool        default=false
            --json              bool        default=false
```
