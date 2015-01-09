// Copyright 2015 James Cote and Liberty Fund, Inc.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package batch_test

import (
	"github.com/jfcote87/google-api-go-client/batch"
	"github.com/jfcote87/google-api-go-client/batch/credentials"
	"log"
	"net/http"

	cal "google.golang.org/api/calendar/v3"
	gmail "google.golang.org/api/gmail/v1"
	storage "google.golang.org/api/storage/v1"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
)

type EventFromDb struct {
	Id          string `json:"id"`
	Title       string `json:"nm"`
	Start       string `json:"start"`
	End         string `json:"end"`
	Description string `json:"desc"` // full description
	Link        string `json:"link"` // web link to full event info
	Loc         string `json:"loc"`  // Event location
}

func ExampleService_Calendar(calendarId string, events []*EventFromDb, oauthClient *http.Client) error {
	// Read through slice of EventFromDb and add to batch.  Then call Do() to send
	// and process responses
	bsv := batch.Service{Client: oauthClient}
	calsv, _ := cal.New(batch.BatchClient)

	for _, ev := range events {
		event := &cal.Event{
			Summary:            ev.Title,
			Description:        ev.Description,
			Location:           ev.Loc,
			Start:              &cal.EventDateTime{DateTime: ev.Start},
			End:                &cal.EventDateTime{DateTime: ev.End},
			Reminders:          &cal.EventReminders{UseDefault: false},
			Transparency:       "transparent",
			Source:             &cal.EventSource{Title: "Web Link", Url: ev.Link},
			ExtendedProperties: &cal.EventExtendedProperties{Shared: map[string]string{"DBID": ev.Id}},
		}
		event, err := calsv.Events.Insert(calendarId, event).Do()
		if err = bsv.AddRequest(err, batch.SetResult(&event), batch.SetTag(ev)); err != nil {
			log.Println(err)
			return err
		}
	}

	responses, err := bsv.Do()
	if err != nil {
		log.Println(err)
		return err
	}
	for _, r := range responses {
		var event *cal.Event
		tag := r.Tag.(*EventFromDb)
		if r.Err != nil {
			log.Printf("Error adding event (Id: %s %s): %v", tag.Id, tag.Title, r.Err)
			continue
		}
		event = r.Result.(*cal.Event)
		updateDatabaseorSomethingElse(tag, event.Id)
	}
	return nil
}

func updateDatabaseorSomethingElse(ev *EventFromDb, newCalEventId string) {
	// Logic for database update or post processing
	return
}

func ExampleService_UserData(projectId string, usernames []string, config *jwt.Config) error {
	// Retrieve the list of available buckets for each user for a given api project as well as
	// profile info for each person
	bsv := batch.Service{} // no need for client as individual requests will have their own authorization
	storagesv, _ := storage.New(batch.BatchClient)
	gsv, _ := gmail.New(batch.BatchClient)
	config.Scopes = []string{gmail.MailGoogleComScope, "email", storage.DevstorageRead_onlyScope}

	for _, u := range usernames {
		// create new credentials for specific user
		tConfig := *config
		tConfig.Subject = u + "@example.com"
		cred := &credentials.Oauth2Credentials{TokenSource: tConfig.TokenSource(oauth2.NoContext)}

		// create bucket list request
		bucketList, err := storagesv.Buckets.List(projectId).Do()
		if err = bsv.AddRequest(err,
			batch.SetResult(&bucketList),
			batch.SetTag([]string{u, "Buckets"}),
			batch.SetCredentials(cred)); err != nil {
			log.Printf("Error adding bucklist request for %s: %v", u, err)
			return err
		}

		// create profile request
		profile, err := gsv.Users.GetProfile(u + "@example.com").Do()
		if err = bsv.AddRequest(err,
			batch.SetResult(&profile),
			batch.SetTag([]string{u, "Profile"}),
			batch.SetCredentials(cred)); err != nil {
			log.Printf("Error adding profile request for %s: %v", u, err)
			return err
		}
	}

	responses, err := bsv.Do()
	if err != nil {
		log.Println(err)
		return err
	}
	// process responses
	for _, r := range responses {
		tag := r.Tag.([]string)
		if r.Err != nil {
			log.Printf("Error retrieving user (%s) %s: %v", tag[0], tag[1], r.Err)
			continue
		}
		if tag[1] == "Profile" {
			profile := r.Result.(*gmail.Profile)
			log.Printf("User %s profile id is %d", profile.EmailAddress, profile.HistoryId)
		} else {
			blist := r.Result.(*storage.Buckets)
			log.Printf("User: %s", tag[0])
			for _, b := range blist.Items {
				log.Printf("%s", b.Name)
			}
		}
	}
	return nil
}
