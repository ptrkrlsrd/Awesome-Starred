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

func (a StarredRepositories) Len() int           { return len(a) }
func (a StarredRepositories) Less(i, j int) bool { return a[i].StarredAt.After(a[j].StarredAt.Time) }
func (a StarredRepositories) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

type StarChannel chan ([]*github.StarredRepository)

func (s StarChannel) Listen(starred *StarredRepositories) {
	for {
		*starred = append(*starred, <-s...)
	}
}
func main() {
	token := os.Getenv("GITHUB_TOKEN")
	ctx, client := newGithubClient(token)

	var starred StarredRepositories
	var starChan = make(StarChannel)

	go starChan.Listen(&starred)

	starList, initialResp, err := getStarsForPage(1, client, ctx)
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

	sort.Sort(StarredRepositories(starred))

	file, err := createFile("README.md")
	if err != nil {
		log.Panic(err)
	}

	w := bufio.NewWriter(file)
	if err := writeStringToBuffer(w, "# Awesome automated list of my starred repositories\n"); err != nil {

		log.Panic(err)
	}

	err = writeStarsToFile(starred, w)
	if err != nil {
		log.Panic(err)
	}

	w.Flush()
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

func writeStarsToFile(starred []*github.StarredRepository, fileWriter *bufio.Writer) error {
	for _, v := range starred {
		name := *v.Repository.Name
		desc := ""
		if v.Repository.Description != nil {
			desc = *v.Repository.Description
		}
		url := *v.Repository.HTMLURL

		content := fmt.Sprintf("\n* [%s](%s) - %s", name, url, desc)
		_, err := fileWriter.WriteString(content)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeStringToBuffer(w *bufio.Writer, content string) error {
	_, err := w.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

func createFile(fileName string) (*os.File, error) {
	f, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}

	return f, err
}

func getStarsForPage(page int, client *github.Client, ctx context.Context) ([]*github.StarredRepository, *github.Response, error) {
	opts := &github.ActivityListStarredOptions{
		Sort:      "created",
		Direction: "desc",
	}

	listOptions := github.ListOptions{Page: page, PerPage: 100}

	opts.ListOptions = listOptions
	starList, resp, err := client.Activity.ListStarred(ctx, "", opts)
	if err != nil {
		return nil, nil, err
	}

	return starList, resp, nil
}
