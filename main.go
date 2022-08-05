package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"

	"golang.org/x/oauth2"

	"github.com/google/go-github/v33/github"
)

type StarredRepositories []*github.StarredRepository

func (starList StarredRepositories) Len() int {
	return len(starList)
}

func (starList StarredRepositories) Less(i, j int) bool {
	return starList[i].StarredAt.After(starList[j].StarredAt.Time)
}

func (starList StarredRepositories) Swap(i, j int) {
	starList[i], starList[j] = starList[j], starList[i]
}

func (starList StarredRepositories) writeAll(writer *bufio.Writer) error {
	for _, v := range starList {
		name := *v.Repository.Name
		url := *v.Repository.HTMLURL

		desc := ""
		if v.Repository.Description != nil {
			desc = *v.Repository.Description
		}

		content := fmt.Sprintf("\n* [%s](%s) - %s", name, url, desc)
		if _, err := writer.WriteString(content); err != nil {
			return err
		}
	}

	return nil
}

func (starList StarredRepositories) SaveToFile(file *os.File) error {
	writer := bufio.NewWriter(file)
	if _, err := writer.WriteString("# Awesome automated list of my starred repositories\n"); err != nil {
		return err
	}

	if err := starList.writeAll(writer); err != nil {
		return err
	}

	writer.Flush()
	return nil
}

type StarChannel chan (StarredRepositories)

func (s StarChannel) Listen(starred *StarredRepositories) {
	for {
		*starred = append(*starred, <-s...)
	}
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	ctx, client := newGithubClient(token)

	var allStars StarredRepositories
	var starChan = make(StarChannel)

	go starChan.Listen(&allStars)

	starList, initialResp, err := getStarsForPage(1, client, ctx)
	if err != nil {
		log.Panic(err)
	}
	starChan <- starList
	maxPages := initialResp.LastPage

	wg := sync.WaitGroup{}
	for i := initialResp.NextPage; i <= maxPages; i++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()

			starList, _, err = getStarsForPage(page, client, ctx)
			if err != nil {
				return
			}

			starChan <- starList
		}(i)
	}
	wg.Wait()

	sort.Sort(StarredRepositories(allStars))

	file, err := os.Create("README.md")
	if err != nil {
		log.Panic(err)
	}

	if err := allStars.SaveToFile(file); err != nil {
		log.Panic(err)
	}
}

func newGithubClient(token string) (context.Context, *github.Client) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return ctx, client
}

func getStarsForPage(page int, client *github.Client, ctx context.Context) ([]*github.StarredRepository, *github.Response, error) {
	opts := &github.ActivityListStarredOptions{Sort: "created", Direction: "desc"}

	opts.ListOptions = github.ListOptions{Page: page, PerPage: 100}
	starList, resp, err := client.Activity.ListStarred(ctx, "", opts)
	if err != nil {
		return nil, nil, err
	}

	return starList, resp, nil
}
