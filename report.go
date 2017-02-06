package main

import (
	"fmt"
	"github.com/urfave/cli"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	FORMAT_TIME  = "01/02/2006"
	FORMAT_MONTH = "01/2006"
)

func print_report_chats(c *cli.Context) error {
	if db == nil {
		log.Fatalln("Database is not initialised")
	}
	mdbObject := CopyDatabase(db)
	defer mdbObject.Close()
	mdb := db.Database

	var results []Chat
	if err := mdb.C(COL_THREADS).Find(nil).All(&results); err != nil {
		log.Fatalln("could not get records", err)
	}
	fmt.Println("id,url,date,month,elapse_time(hours),time_to_first(hours),interactions(total),customer_interactions,support_interactions,interaction_ratio,word_ratio,user_last," +
		"person,channel,withlink,withlinktodoc,totalword,support_level")
	for _, chat := range results {
		line := []string{chat.ChatId, chat.Url}

		//start time
		{
			start := time.Unix(chat.Start, 0)
			line = append(line, start.Format(FORMAT_TIME))
		}

		//start month
		{
			start := time.Unix(chat.Start, 0)
			line = append(line, start.Format(FORMAT_MONTH))
		}
		//elapse time
		{
			total := (chat.End - chat.Start) / (60 * 60)
			line = append(line, strconv.FormatInt(total, 10))
		}

		//time to first
		{
			total := chat.FirstResponser - chat.Start
			if total < 0 {
				total = -1
			} else {
				total = HoursLapseWorkingDays(chat.Start, chat.FirstResponser)
			}
			line = append(line, strconv.FormatInt(total, 10))
		}

		//Interactions
		line = append(line, strconv.FormatInt(int64(chat.Interactions+chat.CustomerInteractions), 10))
		line = append(line, strconv.FormatInt(int64(chat.CustomerInteractions), 10))
		line = append(line, strconv.FormatInt(int64(chat.Interactions), 10))

		//interaction ratio
		{
			total := float64(chat.Interactions) / float64(chat.CustomerInteractions)
			line = append(line, strconv.FormatFloat(total, 'f', 2, 64))
		}

		//word ratio
		{
			total := float64(chat.SupportWords) / float64(chat.CustomerWords)
			line = append(line, strconv.FormatFloat(total, 'f', 2, 64))
		}

		line = append(line, BoolFormat(chat.CustomerLastResponder))

		//report the last author
		if len(chat.Authors) > 0 {
			line = append(line, chat.Authors[len(chat.Authors)-1])
		} else {
			line = append(line, "none")

		}

		line = append(line, chat.Channel)
		line = append(line, BoolFormat(chat.RespondedWithLink))
		line = append(line, BoolFormat(chat.RespondidWithDoc))

		//total wordcount
		line = append(line, strconv.FormatInt(int64(chat.SupportWords+chat.CustomerWords), 10))

		//Estimated level of support
		//TODO
		{
			level := "level2"
			if chat.SupportWords+chat.CustomerWords > 1000 || chat.Interactions+chat.CustomerInteractions > 10 {
				level = "level3"
			}
			if chat.SupportWords+chat.CustomerWords < 200 || chat.Interactions+chat.CustomerInteractions < 3 {
				level = "level1"
			}
			line = append(line, level)
		}
		fmt.Println(strings.Join(line, ","))

	}
	return nil
}

func BoolFormat(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func HoursLapseWorkingDays(start int64, end int64) int64 {
	startTime := time.Unix(start, 0)

	if startTime.Weekday() == time.Saturday {
		newStart := startTime.Add(time.Hour * 48)
		year, month, day := newStart.Date()
		startTime = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	} else if startTime.Weekday() == time.Sunday {
		newStart := startTime.Add(time.Hour * 24)
		year, month, day := newStart.Date()
		startTime = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
	return (end - startTime.Unix()) / 3600
}
