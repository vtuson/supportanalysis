package main

import (
	"errors"
	"gopkg.in/mgo.v2/bson"
	"log"
	"regexp"
	"strings"
	"time"
)

type Entry struct {
	Id        bson.ObjectId `bson:"_id,omitempty"`
	ChatId    string        `bson:"chatId"`
	Author    string        `bson:"author"`
	Title     string        `bson:"title"`
	Url       string        `bson:"url"`
	Timestamp int64         `bson:"timestamp"`
	Reply     string        `bson:"reply"`
	Channel   string        `bson:"channel"`
}

func (e *Entry) WordCount() int {
	re := regexp.MustCompile("\\w+")
	return len(re.FindAllString(e.Reply, -1))
}
func (e *Entry) IsCustomer() bool {
	return e.Author == CUSTOMER_ID
}

func (e *Entry) HasDocLink() bool {
	return strings.Contains(e.Reply, "docs.bitnami.com")
}
func (e *Entry) HasLink() bool {
	re := regexp.MustCompile("http[s]://\\w+")
	return re.MatchString(e.Reply)
}

type Chat struct {
	Id                    bson.ObjectId `bson:"_id,omitempty"`
	ChatId                string        `bson:"chatId"`
	Authors               []string      `bson:"authors"`
	Title                 string        `bson:"title"`
	Url                   string        `bson:"url"`
	Start                 int64         `bson:"startTime"`
	End                   int64         `bson:"endTime"`
	FirstResponser        int64         `bson:"firstResponseTime"`
	Interactions          int           `bson:"interactionsSupport"`
	CustomerInteractions  int           `bson:"interactionsCustomer"`
	CustomerWords         int           `bson:"customerWords"`
	SupportWords          int           `bson:"supportWords"`
	CustomerLastResponder bool          `bson:"customerLast"`
	Channel               string        `bson:"channel"`
	RespondedWithLink     bool          `bson:"hasLink"`
	RespondidWithDoc      bool          `bson:"hasDoc"`
}

func (c *Chat) Update() error {
	if db == nil {
		log.Fatalln("Database is not initialised")
	}
	mdbObject := CopyDatabase(db)
	defer mdbObject.Close()
	mdb := db.Database
	return mdb.C(COL_THREADS).Update(bson.M{"_id": c.Id}, c)
}

func (c *Chat) addAuthor(author string) {
	for _, a := range c.Authors {
		if strings.Compare(a, author) == 0 {
			return
		}
	}
	c.Authors = append(c.Authors, author)
}

func (c *Chat) UpdateEntry(ent *Entry) {
	timeStart := time.Unix(c.Start, 0)
	timeEnd := time.Unix(c.End, 0)
	firstResponse := time.Unix(c.FirstResponser, 0)
	entTime := time.Unix(ent.Timestamp, 0)

	if timeStart.After(entTime) {
		c.Start = ent.Timestamp
	}
	if timeEnd.Before(entTime) {
		c.End = ent.Timestamp
		c.CustomerLastResponder = ent.IsCustomer()
	}

	if ent.IsCustomer() {
		c.CustomerInteractions++
		c.CustomerWords += ent.WordCount()
	} else {
		c.SupportWords += ent.WordCount()
		c.Interactions++
		c.addAuthor(ent.Author)
		if c.FirstResponser == 0 {
			c.FirstResponser = ent.Timestamp
		} else {
			if firstResponse.After(entTime) {
				c.FirstResponser = ent.Timestamp
			}
		}
		if !c.RespondedWithLink {
			c.RespondedWithLink = ent.HasLink()
		}
		if !c.RespondidWithDoc {
			c.RespondidWithDoc = ent.HasDocLink()
		}

	}
}

func addEntry(line string) (*Entry, error) {
	if db == nil {
		log.Fatalln("Database is not initialised")
	}
	mdbObject := CopyDatabase(db)
	defer mdbObject.Close()
	mdb := db.Database

	l := strings.Split(line, "\t")
	if len(l) < 5 {
		log.Println("line error formating", len(l), line)
		return nil, errors.New("line error formating")
	}

	//create entry
	ent := &Entry{
		Id:     bson.NewObjectId(),
		Title:  l[0],
		Author: strings.TrimSpace(l[2]),
		Url:    strings.TrimSpace(l[1]),
	}

	re := regexp.MustCompile("\\[(.*?)\\]")
	re_result := re.FindStringSubmatch(l[0])
	if len(re_result) < 2 {
		log.Fatalln("could not proccess id", ent.Title)
	}
	ent.ChatId = re_result[1]

	if t, err := time.Parse(TIMEFORMAT, l[3]); err == nil {
		ent.Timestamp = t.Unix()
	} else {
		log.Println("line error with time formating", line, err)
		return nil, err
	}
	ent.Reply = strings.Join(l[4:], " ")

	if strings.Contains(ent.ChatId, "C") {
		ent.Channel = CHANNEL_COMMUNITY
	} else {
		ent.Channel = CHANNEL_ZENDESK
	}

	log.Println("saving entry", ent.Title)

	//update chat
	chat := getChat(ent)
	if chat == nil {
		return nil, nil
	}
	chat.UpdateEntry(ent)
	chat.Update()

	return ent, mdb.C(COL_ENTRIES).Insert(ent)

}

//creates a new chat from an entry
func getChat(e *Entry) *Chat {
	if c := findChatById(e.ChatId); c != nil {
		return c
	}
	if !e.IsCustomer() {
		log.Println("author of chat is not a customer and chat does not exist", e.Author)
		return nil
	}
	if db == nil {
		log.Fatalln("Database is not initialised")
	}
	mdbObject := CopyDatabase(db)
	defer mdbObject.Close()
	mdb := db.Database

	c := &Chat{
		Id:      bson.NewObjectId(),
		Title:   e.Title,
		Start:   e.Timestamp,
		End:     e.Timestamp,
		Url:     e.Url,
		ChatId:  e.ChatId,
		Channel: e.Channel,
	}
	if err := mdb.C(COL_THREADS).Insert(c); err != nil {
		log.Fatalln("could not insert new chat", err)
	}
	return c
}

func findChatById(id string) *Chat {
	if db == nil {
		log.Fatalln("Database is not initialised")
	}
	mdbObject := CopyDatabase(db)
	defer mdbObject.Close()
	mdb := db.Database

	find := bson.M{"chatId": id}
	q := mdb.C(COL_THREADS).Find(find)
	if q == nil {
		log.Fatalln("error creating query", id)
	}
	count, err := q.Count()
	if err != nil {
		log.Fatalln("could not count chat entries for", id)
	}
	if count > 1 {
		log.Fatalln("multiple entries for", id)
	}
	if count < 1 {
		return nil
	}
	var result Chat
	if err := q.One(&result); err != nil {
		log.Fatalln("failed to retrieved chat", id, err)
	}
	return &result
}

func Count_Entries() int {
	if db == nil {
		log.Fatalln("Database is not initialised")
	}
	mdbObject := CopyDatabase(db)
	defer mdbObject.Close()
	mdb := db.Database
	if res, err := mdb.C(COL_ENTRIES).Count(); err != nil {
		log.Fatalln(err)
	} else {
		return res
	}
	return -1
}

func Count_Chats() int {
	if db == nil {
		log.Fatalln("Database is not initialised")
	}
	mdbObject := CopyDatabase(db)
	defer mdbObject.Close()
	mdb := db.Database
	if res, err := mdb.C(COL_THREADS).Count(); err != nil {
		log.Fatalln(err)
	} else {
		return res
	}
	return -1

}
