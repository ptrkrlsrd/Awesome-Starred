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

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var starred []*github.StarredRepository
	var starChan = make(chan []*github.StarredRepository)

	go func() {
		for {
			stars := <-starChan
			starred = append(starred, stars...)
		}
	}()

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

	log.Printf("Total stars: %d", len(starred))
	sort.Sort(StarredRepositories(starred))

	file, err := createFile("./README.md")
	if err != nil {
		log.Panic(err)
	}

	fileWriter, err := writeStringToFile(file, "# Awesome - Starred repositories")
	if err != nil {
		log.Panic(err)
	}

	for _, v := range starred {
		name := *v.Repository.Name
		desc := ""
		if v.Repository.Description != nil {
			desc = *v.Repository.Description
		}
		url := *v.Repository.HTMLURL

		content := fmt.Sprintf("\n* [%s](%s) - %s", name, url, desc)
		_, err = fileWriter.WriteString(content)
		if err != nil {
			panic(err)
		}
	}

	fileWriter.Flush()
}

func createFile(fileName string) (*os.File, error) {
	f, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}

	return f, err
}

func writeStringToFile(f *os.File, content string) (*bufio.Writer, error) {
	w := bufio.NewWriter(f)
	_, err := w.WriteString(content)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func getStarsForPage(page int, client *github.Client, ctx context.Context) ([]*github.StarredRepository, *github.Response, error) {
	opts := &github.ActivityListStarredOptions{
		Sort:      "created",
		Direction: "desc",
	}

	listOptions := github.ListOptions{
		Page:    page,
		PerPage: 100,
	}

	opts.ListOptions = listOptions
	starList, initialResp, err := client.Activity.ListStarred(ctx, "", opts)
	if err != nil {
		return nil, nil, err
	}

	return starList, initialResp, nil
}
