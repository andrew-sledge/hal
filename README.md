# hal
Read from a redis list and post to Slack. You know, all that chatops stuff.

## Usage
```
./hal	--redis_connection  string      default="127.0.0.1:6379"	The connection string to the redis cluster
        --redis_db          int         default=0					The redis database number
        --redis_list        string      default="mylist"			The redis database list
        --slack_channel     string      default=NONE (REQUIRED)		The slack channel - the name
        --slack_token       string      default=NONE (REQUIRED)		Get this from your integration
        --watch_interval    int         default=3					In seconds, how often to pull from the list
        --debug             bool        default=false				Turn on verbose
        --json              bool        default=false				Expect JSON data
```

## TODO
* ~~Make channel lookup "smart"~~
