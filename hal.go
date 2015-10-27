package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/fzzy/radix/extra/cluster"
	"github.com/fzzy/radix/redis"
	"github.com/nlopes/slack"
	"os"
	"strconv"
	"strings"
	"time"
)

type Opts struct {
	redis_connection string
	redis_db         int
	redis_list       string
	slack_channel    string
	slack_token      string
	watch_interval   int
	debug            bool
	json             bool
}

func pull(rtm *slack.RTM, o Opts) {
	/*
		Using a FIFO queue
	*/
	t := time.Now()
	ts := t.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
	rc, err := cluster.NewCluster(o.redis_connection)

	if err != nil {
		fmt.Printf("[%s] ERROR Redis connection error: %s\n", ts, err)
		os.Exit(1)
	}
	r := rc.Cmd("SELECT", o.redis_db)

	for {
		t = time.Now()
		ts = t.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
		r = rc.Cmd("RPOP", o.redis_list)

		switch r.Type {
		case redis.ErrorReply:
			fmt.Printf("[%s] ERROR ErrorReply received: %s\n", ts, r.Err.Error())
		case redis.NilReply:
			if o.debug {
				fmt.Printf("[%s] INFO NilReply reply received\n", ts)
			}
		case redis.StatusReply:
			if o.debug {
				fmt.Printf("[%s] INFO StatusReply reply received: not processing\n", ts)
			}
		case redis.BulkReply:
			// Send to Slack
			data, err := r.Bytes()
			if err != nil {
				fmt.Printf("[%s] ERROR Error received: %s\n", ts, err)
			} else {
				if o.json {
					type Message struct {
						Name   string
						Source string
						Detail string
					}
					var message Message
					err := json.Unmarshal(data, &message)
					if err != nil {
						fmt.Printf("[%s] ERROR Error received: %s\n", ts, err)
					}
					params := slack.PostMessageParameters{
						Username: "hal9000",
					}
					attachment := slack.Attachment{
						Pretext: message.Source,
						Text:    message.Detail,
					}
					params.Attachments = []slack.Attachment{attachment}

					channelID, timestamp, err := rtm.PostMessage(o.slack_channel, string(message.Name), params)
					if err != nil {
						fmt.Printf("[%s] ERROR Error received: %s\n", ts, err)
						return
					}
					if o.debug {
						fmt.Printf("[%s] INFO Message %+v successfully sent to channel %s at %s", ts, message, channelID, timestamp)
					}
				} else {
					if o.debug {
						fmt.Printf("[%s] INFO BulkReply reply received: %s\n", ts, data)
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(string(data), o.slack_channel))
				}
			}
		case redis.MultiReply:
			if o.debug {
				fmt.Printf("[%s] INFO MultiReply reply received: not processing\n", ts)
			}
		case redis.IntegerReply:
			if o.debug {
				fmt.Printf("[%s] INFO IntegerReply reply received: not processing\n", ts)
			}
		default:
			if o.debug {
				fmt.Printf("[%s] INFO Unknown reply received: not processing\n", ts)
			}
		}

		time.Sleep(time.Duration(o.watch_interval) * time.Millisecond)
	}
}

func get_channel(rtm *slack.RTM, lookfor string) (string, error) {
	has_hash := strings.HasPrefix(strings.Trim(lookfor, " "), "#")
	if has_hash {
		lookfor = strings.TrimPrefix(strings.Trim(lookfor, " "), "#")
	}
	l, err := rtm.GetChannels(false)
	if err != nil {
		t := time.Now()
		ts := t.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
		fmt.Printf("[%s] ERROR Channel list error: %s\n", ts, err)
		os.Exit(1)
	}
	for _, v := range l {
		if v.Name == lookfor {
			return v.ID, nil
		}
	}
	return "", errors.New("No channel found with this name")
}

func talk(rtm *slack.RTM, o Opts) {
	t := time.Now()
	ts := t.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
Loop:
	for {
		t = time.Now()
		ts = t.Format("Mon Jan 2 15:04:05 -0700 MST 2006")
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				if o.debug {
					fmt.Printf("[%s] INFO Received hello: %v\n", ts, ev)
				}

			case *slack.ConnectedEvent:
				if o.debug {
					fmt.Printf("[%s] INFO Event information: %s\n", ts, ev.Info)
				}
				rtm.SendMessage(rtm.NewOutgoingMessage("Hello, Dave", o.slack_channel))

			case *slack.MessageEvent:
				if o.debug {
					fmt.Printf("[%s] INFO Message event information: %v\n", ts, ev)
				}

			case *slack.PresenceChangeEvent:
				if o.debug {
					fmt.Printf("[%s] INFO Presence change: %v\n", ts, ev)
				}

			case *slack.LatencyReport:
				if o.debug {
					fmt.Printf("[%s] INFO Current latency: %v\n", ts, ev.Value)
				}

			case *slack.RTMError:
				fmt.Printf("[%s] ERROR Slack RTM Error: %s\n", ts, ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("[%s] ERROR Invalid credentials\n", ts)
				break Loop

			default:
				if o.debug {
					fmt.Printf("[%s] INFO Unexpected event: %v\n", ts, msg.Data)
				}
			}
		}
	}
}

func main() {

	// Set up default values
	o := Opts{
		redis_connection: "127.0.0.1:6379",
		redis_db:         0,
		redis_list:       "mylist",
		watch_interval:   3,
		debug:            false,
		json:             false,
	}

	t := time.Now()
	ts := t.Format("Mon Jan 2 15:04:05 -0700 MST 2006")

	// Grab flags, but OS environment vars take precedence
	rc_e, err := os.LookupEnv("REDIS_CONNECTION")
	if err == false {
		flag.StringVar(&o.redis_connection, "redis_connection", "127.0.0.1:6379", "Redis connection string")
	} else {
		o.redis_connection = rc_e
	}

	rd_e, err := os.LookupEnv("REDIS_DB")
	if err == false {
		flag.IntVar(&o.redis_db, "redis_db", 0, "Redis database number")
	} else {
		rd, errb := strconv.Atoi(rd_e)
		if errb != nil {
			fmt.Printf("[%s] ERROR Specified Redis DB unrecognizable\n", ts)
			os.Exit(1)
		}
		o.redis_db = rd
	}

	rl_e, err := os.LookupEnv("REDIS_LIST")
	if err == false {
		flag.StringVar(&o.redis_list, "redis_list", "mylist", "Redis list name")
	} else {
		o.redis_list = rl_e
	}

	wi_e, err := os.LookupEnv("WATCH_INTERVAL")
	if err == false {
		flag.IntVar(&o.watch_interval, "watch_interval", 3, "Watch interval for pulling from Redis list.")
	} else {
		wi, errb := strconv.Atoi(wi_e)
		if errb != nil {
			fmt.Printf("[%s] ERROR Specified watch interval unrecognizable\n", ts)
			os.Exit(1)
		}
		o.watch_interval = wi
	}

	var sc_e string
	sc_e_temp, err := os.LookupEnv("SLACK_CHANNEL")
	if err == false {
		flag.StringVar(&sc_e, "slack_channel", "", "Slack channel (required)")
	} else {
		sc_e = sc_e_temp
	}

	st_e, err := os.LookupEnv("SLACK_TOKEN")
	if err == false {
		flag.StringVar(&o.slack_token, "slack_token", "", "Slack token (required)")
	} else {
		o.slack_token = st_e
	}

	dg_e, err := os.LookupEnv("HAL_DEBUG")
	if err == false {
		flag.BoolVar(&o.debug, "debug", false, "Turn on debug")
	} else {
		dg, errb := strconv.ParseBool(dg_e)
		if errb != nil {
			fmt.Printf("[%s] ERROR Specified debug flag value unrecognizable\n", ts)
			os.Exit(1)
		}
		o.debug = dg
	}

	js_e, err := os.LookupEnv("HAL_JSON")
	if err == false {
		flag.BoolVar(&o.json, "json", false, "Message will be in JSON format")
	} else {
		js, errb := strconv.ParseBool(js_e)
		if errb != nil {
			fmt.Printf("[%s] ERROR Specified json flag value unrecognizable\n", ts)
			os.Exit(1)
		}
		o.json = js
	}

	flag.Parse()

	api := slack.New(o.slack_token)
	if o.debug {
		api.SetDebug(true)
	}

	rtm := api.NewRTM()
	go rtm.ManageConnection()

	my_channel, errc := get_channel(rtm, sc_e)
	if errc != nil {
		fmt.Printf("[%s] ERROR %s\n", ts, errc.Error())
		os.Exit(1)
	}
	o.slack_channel = my_channel

	if len(o.slack_token) == 0 || len(o.slack_channel) == 0 {
		fmt.Printf("[%s] ERROR Slack Token or Slack Channel not specified\n", ts)
		os.Exit(1)
	} else {
		fmt.Printf("[%s] Slack Token: %s, Slack Channel: %s\n", ts, o.slack_token, o.slack_channel)
	}

	go pull(rtm, o)
	talk(rtm, o)
}
