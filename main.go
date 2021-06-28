package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"golang.org/x/oauth2"

	"github.com/google/go-github/v33/github"
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var starred []*github.StarredRepository
	opts := &github.ActivityListStarredOptions{
		Sort:      "created",
		Direction: "desc",
	}

	var starChan = make(chan []*github.StarredRepository)

	go func() {
		for {
			stars := <-starChan
			starred = append(starred, stars...)
			log.Printf("stars: %d", len(starred))
		}
	}()

	listOptions := github.ListOptions{
		Page:    0,
		PerPage: 100,
	}

	opts.ListOptions = listOptions

	starList, resp, err := client.Activity.ListStarred(ctx, "", opts)
	if err != nil {
		return
	}
	maxPages := resp.LastPage
	log.Println(resp.LastPage)
	starChan <- starList

	wg := sync.WaitGroup{}

	for i := resp.NextPage; i <= maxPages; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			listOptions := github.ListOptions{
				Page:    page,
				PerPage: 100,
			}

			opts.ListOptions = listOptions
			starList, resp, err = client.Activity.ListStarred(ctx, "", opts)
			if err != nil {
				return
			}

			if len(starList) == 0 {
				return
			}

			starChan <- starList
		}(i)
	}
	wg.Wait()

	log.Println(len(starred))

	f, err := os.Create("./README.md")
	if err != nil {
		panic(err)
	}

	w := bufio.NewWriter(f)
	_, err = w.WriteString("# Awesome - Starred repositories\n")
	if err != nil {
		panic(err)
	}

	for _, v := range starred {
		name := *v.Repository.Name
		desc := ""
		if v.Repository.Description != nil {
			desc = *v.Repository.Description
		}
		url := *v.Repository.HTMLURL

		content := fmt.Sprintf("* [%s](%s) - %s\n", name, url, desc)
		_, err = w.WriteString(content)
		if err != nil {
			panic(err)
		}
	}

	w.Flush()
}
