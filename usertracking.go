package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type UserInfo struct {
	IP          string
	Name        string
	Passwords   []string
	LastAttempt time.Time
}

type activityEntry struct {
	user      string
	ip        string
	action    string
	timestamp time.Time
}

func logActivity(ip string, user string, action string) {
	ac := activityEntry{ip: ip, user: user, action: action, timestamp: time.Now()}
	activityMu.Lock()
	activity = append(activity, ac)
	activityMu.Unlock()
	fmt.Println(ac.timestamp.String() + " " + ac.ip + ":" + ac.user + " - " + ac.action)
}

func logLoginAttempt(ip string, user string, password string) {
	for k, v := range users {
		if v.IP == ip && v.Name == user {
			found := false
			for _, v := range v.Passwords {
				if v == password {
					found = true
					break
				}
			}
			if !found {
				users[k].Passwords = append(users[k].Passwords, password)
			}
			users[k].LastAttempt = time.Now()
			return
		}
	}
	users = append(users, UserInfo{IP: ip, Name: user, Passwords: []string{password}, LastAttempt: time.Now()})
}

// We don't want attackers to be able to abuse our disk IO by DOSing us with activity.
// The idea is that this process will handle getting that activity logged on its own schedule.
func writeActivityToDisk() {
	for {
		file, err := os.OpenFile(activityLogPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err == nil {
			if len(activity) > 0 {
				activityMu.Lock()
				//Write activity to disk.
				for _, ac := range activity {
					file.WriteString(ac.timestamp.String() + " - IP:" + ac.ip + " USER: " + ac.user + " ACTION:" + ac.action + "\n")
				}

				//Empty activity array.
				activity = make([]activityEntry, 0)
				activityMu.Unlock()
			}

			file.Close()

			userFile, _ := json.MarshalIndent(users, "", " ")
			_ = ioutil.WriteFile(userLogPath, userFile, 0644)

		} else {
			log.Fatal(err)
		}

		time.Sleep(5 * time.Second)
	}
}
