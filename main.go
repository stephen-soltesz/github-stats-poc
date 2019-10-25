// Copyright 2017 github-label-sync Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//////////////////////////////////////////////////////////////////////////////

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	fAuthtoken   string
	fGithubOwner string
	fGithubRepo  string
)

const (
	usage = `
Usage of %s:

Github receiver requires a github --authtoken and target github --owner and
--repo names.

`
)

func init() {
	flag.StringVar(&fAuthtoken, "authtoken", "", "Oauth2 token for access to github API.")
	flag.StringVar(&fGithubOwner, "owner", "", "The github user or organization name.")
	flag.StringVar(&fGithubRepo, "repo", "", "The repository where issues are created.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
	}
}

// A Client manages communication with the Github API.
type Client struct {
	// githubClient is an authenticated client for accessing the github API.
	GithubClient *github.Client
	// owner is the github project (e.g. github.com/<owner>/<repo>).
	owner string
	// repo is the github repository under the above owner.
	repo string
}

// NewClient creates an Client authenticated using the Github authToken.
// Future operations are only performed on the given github "owner/repo".
func NewClient(owner, repo, authToken string) *Client {
	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: authToken},
	)
	client := &Client{
		GithubClient: github.NewClient(oauth2.NewClient(ctx, tokenSource)),
		owner:        owner,
		repo:         repo,
	}
	return client
}

func pString(s string) *string {
	return &s
}

type localClient Client

var repoNames = []string{}

func main() {
	flag.Parse()
	if fAuthtoken == "" || fGithubOwner == "" || fGithubRepo == "" {
		flag.Usage()
		os.Exit(1)
	}
	client := (*localClient)(NewClient(fGithubOwner, fGithubRepo, fAuthtoken))

	//opt := &github.RepositoryListByOrgOptions{
	//ListOptions: github.ListOptions{PerPage: 20},
	//}

	ctx := context.Background()

	// get all pages of results
	/*
		fmt.Println("Getting Repos")
		var allRepos []*github.Repository
		for {
			repos, resp, err := client.GithubClient.Repositories.ListByOrg(ctx, "m-lab", opt)
			if err != nil {
				panic(err)
			}
			allRepos = append(allRepos, repos...)
			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}

		for _, repo := range allRepos {
			// pretty.Print(repo)
			repoNames = append(repoNames, repo.GetName())
		}

		pretty.Print(repoNames)
	*/
	opt2 := &github.PullRequestListOptions{
		State:       "closed",
		ListOptions: github.ListOptions{PerPage: 50},
	}

	// var allPRs []*github.PullRequest

	fmt.Println("Getting PRs")
	repoNames = []string{"prometheus-support"}
	for _, repo := range repoNames {
		fmt.Println(repo)
		for {
			prs, prResp, err := client.GithubClient.PullRequests.List(ctx, fGithubOwner, repo, opt2)
			if err != nil {
				panic(err)
			}
			for _, pr := range prs {
				// NOTE: pr.RequestedReviewers only contains reviewers who have not approved the PR, whereas ListReviews includes all users.
				reviews, resp, err := client.GithubClient.PullRequests.ListReviews(ctx, fGithubOwner, pr.GetBase().GetRepo().GetName(), pr.GetNumber(), &github.ListOptions{PerPage: 2})
				if err != nil {
					fmt.Println(pr.GetBase().GetRepo().GetName(), pr.GetNumber())
					panic(err)
				}
				for _, review := range reviews {
					fmt.Println(resp.Rate.Remaining, pr.GetBase().GetRepo().GetName(), pr.GetNumber(), pr.GetUser().GetLogin(), review.GetUser().GetLogin(), review.GetSubmittedAt().Format(time.RFC3339))
				}
				if resp.Remaining == 0 {
					fmt.Println("Sleeping until:", time.Until(resp.Rate.Reset.Time))
					time.Sleep(time.Until(resp.Rate.Reset.Time))
				}
			}
			if prResp.NextPage == 0 {
				break
			}
			opt2.Page = prResp.NextPage
		}
	}

}
