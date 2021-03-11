package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v33/github"
)

func main() {
	client := github.NewClient(nil)

	var starred []*github.StarredRepository
	opts := &github.ActivityListStarredOptions{
		Sort:      "created",
		Direction: "desc",
	}

	total := 0

	for i := 1; i < 1000; i++ {
		listOptions := github.ListOptions{
			Page:    i,
			PerPage: 100,
		}

		opts.ListOptions = listOptions

		starList, _, err := client.Activity.ListStarred(context.Background(), "ptrkrlsrd", opts)
		if err != nil {
			return
		}

		total += len(starList)

		if len(starList) == 0 {
			break
		}

		fmt.Println(total)

		starred = append(starred, starList...)
	}

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
		if *&v.Repository.Description != nil {
			desc = *v.Repository.Description
		}
		url := *v.Repository.URL

		content := fmt.Sprintf("## [%s](%s) - %s\n", name, url, desc)
		_, err = w.WriteString(content)
		if err != nil {
			panic(err)
		}
	}

	w.Flush()
}
