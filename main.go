package main

import (
	"bufio"
	"fmt"
	"github.com/urfave/cli"
	"log"
	"os"
)

const (
	FILE_TO_PROCESS   = "support.tsv"
	COL_ENTRIES       = "entries"
	COL_THREADS       = "chats"
	TIMEFORMAT        = "2006-01-02T15:04:05.999Z"
	CHANNEL_COMMUNITY = "community"
	CHANNEL_ZENDESK   = "zendesk"
	CUSTOMER_ID       = "customer"
)

var db *Database

func init() {
	db = NewDb("support")
	if err := db.Connect("mongodb://localhost:27017/admin"); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "support"
	app.Usage = "imports data from a tsv and analyses support"
	app.Commands = []cli.Command{
		{
			Name:   "import",
			Usage:  "imports data from " + FILE_TO_PROCESS,
			Action: import_data,
		},
		{
			Name:  "count",
			Usage: "count entries",
			Action: func(c *cli.Context) error {
				fmt.Println("chats: ", Count_Chats())
				fmt.Println("entries: ", Count_Entries())
				return nil
			},
		},
		{
			Name:   "print",
			Usage:  "print [cid] -- returns chat with that id",
			Action: print_chat,
		},
		{
			Name:   "chatreport",
			Usage:  "prints out a csv report for chats",
			Action: print_report_chats,
		},
	}
	app.Run(os.Args)
}

func import_data(c *cli.Context) error {
	file, err := os.Open(FILE_TO_PROCESS)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if _, err := addEntry(scanner.Text()); err != nil {
			log.Fatalln(err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return nil
}

func print_chat(c *cli.Context) error {
	var cid string
	if c.NArg() > 0 {
		cid = c.Args().Get(0)
	}
	fmt.Printf("%#v\n", *findChatById(cid))
	return nil
}
