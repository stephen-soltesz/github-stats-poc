package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/m-lab/go/rtx"

	"github.com/google/go-github/github"
	"github.com/stephen-soltesz/github-webhook-receiver/githubx"
	"github.com/stephen-soltesz/pretty"
)

func main() {
	c := githubx.NewClient(0)
	ctx := context.Background()

	opt := &github.SearchOptions{
		Sort:  "created",
		Order: "asc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	query := os.Args[1]
	result, resp, err := c.Search.Issues(ctx, query, opt)
	rtx.Must(err, "Failed to search issues: %s :", query)
	fmt.Println(resp.Rate)

	for _, issue := range result.Issues {
		if false {
			pretty.Print(issue)

		} else {
			fmt.Printf("%-55s %s %3d %-15s %s\n",
				issue.GetHTMLURL(),
				issue.GetCreatedAt().Format(time.RFC3339),
				issue.GetNumber(),
				// issue.GetRepositoryURL(),
				// issue.GetRepository().GetHTMLURL(),
				issue.GetUser().GetLogin(),
				issue.GetTitle(),
			)
		}
	}

	/*
		opt := &github.SearchOptions{}

		sr, _, err := c.Search.Issues(
			ctx, "is:pr user:stephen-soltesz repo:m-lab/prometheus-support",
			opt)
		if err != nil {
			log.Fatal(err)
		}
		// pretty.Print(resp)
		for i := range sr.Issues {
			issue := sr.Issues[i]
			pretty.Print(issue)
			fmt.Printf("%d %-20s %s\n",
				issue.GetNumber(),
				issue.GetUser().GetLogin(),
				issue.GetTitle())
		}
	*/

	/*
		opt := &github.PullRequestListOptions{
			State: "all",
		}
		var allPRs []*github.PullRequest
		for {
			prs, resp, err := c.PullRequests.List(ctx, "m-lab", "prometheus-support", opt)
			if err != nil {
				log.Fatal(err)
			}
			pretty.Print(resp)

			allPRs = append(allPRs, prs...)
			if resp.NextPage == 0 {
				break
			}
			break
			opt.Page = resp.NextPage
		}

		for i := range allPRs {
			// pretty.Print(allPRs[i])
			reviewer := ""
			reviews, _, err := c.PullRequests.ListReviews(
				ctx, "m-lab", "prometheus-support",
				allPRs[i].GetNumber(), nil)
			if err != nil {
				log.Fatal(err)
			}
			for i := range reviews {
				if reviews[i].GetState() == "APPROVED" {
					reviewer += reviews[i].GetUser().GetLogin() + ","
				}
			}
			fmt.Printf("%d %s %-20s %-20s\n",
				allPRs[i].GetNumber(),
				allPRs[i].GetMergedAt(),
				allPRs[i].User.GetLogin(),
				reviewer)
		}

		/*
			pr, _, err := c.PullRequests.Get(ctx, "m-lab", "prometheus-support", 377)
			if err != nil {
				log.Fatal(err)
			}
	*/
}
