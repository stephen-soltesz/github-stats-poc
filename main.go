// Copyright 2019 github-stats-poc Authors
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
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	fAuthtoken   string
	fGithubOwner string
	fGithubRepo  string
	fNumber      int
)

const (
	usage = `
Usage of %s:

Requires a github --authtoken and source github --owner. Optionally provide
the -repo and -num of a single PR to limit results.

`
)

func init() {
	flag.StringVar(&fAuthtoken, "authtoken", "", "Oauth2 token for access to github API.")
	flag.StringVar(&fGithubOwner, "owner", "", "The github user or organization name.")
	flag.StringVar(&fGithubRepo, "repo", "", "Limit search to a single given repository.")
	flag.IntVar(&fNumber, "num", 0, "Limit search to a single given PR number.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
	}
}

// NewClient creates an Client authenticated using the Github authToken.
// Future operations are only performed on the given github "owner/repo".
func NewClient(authToken string) *github.Client {
	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: authToken},
	)
	return github.NewClient(oauth2.NewClient(ctx, tokenSource))
}

func readRepoFile(name string) []string {
	var names []string
	file, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		names = append(names, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return names
}

func writeReposFile(name string, repos []string) {
	fmt.Println("Saving repos to:", name)
	content := strings.Join(repos, "\n")
	b := []byte(content + "\n")
	ioutil.WriteFile(name, b, 0644)
}

func getRepos(client *github.Client) []string {
	if info, _ := os.Stat("repos.txt"); info != nil {
		fmt.Println("Loading cached repos.txt")
		names := readRepoFile("repos.txt")
		return names
	}

	fmt.Println("Listing repos from API")
	ctx := context.Background()
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 20},
	}
	var names []string
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, fGithubOwner, opt)
		if _, ok := err.(*github.RateLimitError); ok {
			log.Printf("rate limit error! %#v\n", resp.Rate)
			if resp.Rate.Remaining == 0 {
				fmt.Println("Sleeping until:", time.Until(resp.Rate.Reset.Time))
				time.Sleep(time.Until(resp.Rate.Reset.Time))
			}
			continue
		}
		if err != nil {
			panic(err)
		}
		for _, repo := range repos {
			names = append(names, repo.GetName())
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	sort.Strings(names)
	writeReposFile("repos.txt", names)
	return names
}

var errRetry = fmt.Errorf("Caller should retry call")

func checkPR(ctx context.Context, client *github.Client, pr *github.PullRequest) ([]string, error) {
	if pr.GetMergedAt().IsZero() {
		// Skip abandoned PRs. PRs that were never merged have a zero merge time.
		return nil, nil
	}
	if pr.GetMergedAt().Before(time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		// Only look at PRs after 2017-01-01.
		return nil, nil
	}
	if fNumber != 0 && fNumber != pr.GetNumber() {
		// Skip all other PRs than the requested one.
		fmt.Println("Skipping:", fNumber, pr.GetNumber())
		return nil, nil
	}
	// NOTE: pr.RequestedReviewers only contains reviewers who have not
	// approved the PR, whereas ListReviews includes completed reviews.
	reviews, resp, err := client.PullRequests.ListReviews(
		ctx, fGithubOwner, pr.GetBase().GetRepo().GetName(),
		pr.GetNumber(), &github.ListOptions{PerPage: 3})
	if _, ok := err.(*github.RateLimitError); ok {
		log.Printf("Rate limit error! %#v\n", resp.Rate)
		if resp.Rate.Remaining == 0 {
			fmt.Println("Sleeping until:", time.Until(resp.Rate.Reset.Time))
			time.Sleep(time.Until(resp.Rate.Reset.Time))
		}
		return nil, errRetry
	}
	if err != nil {
		fmt.Println(pr.GetBase().GetRepo().GetName(), pr.GetNumber())
		panic(err)
	}
	fmt.Println(
		"Checking:", resp.Rate.Remaining, pr.GetBase().GetRepo().GetName(),
		pr.GetNumber(), len(reviews), len(pr.RequestedReviewers))
	prLines := []string{}
	// Before we started using github reviews, we have requested reviews
	// and ":lgtm:" in comments. Here we simply use requested reviewers
	// as proof of review.
	if len(reviews) == 0 && len(pr.RequestedReviewers) == 0 {
		users := map[string]string{}
		comments, resp, err := client.Issues.ListComments(
			ctx, fGithubOwner, pr.GetBase().GetRepo().GetName(), pr.GetNumber(),
			&github.IssueListCommentsOptions{
				// For this case we'll only look once, so get a large number of comments.
				Since:       time.Date(2017, time.January, 1, 0, 0, 0, 0, time.UTC),
				ListOptions: github.ListOptions{PerPage: 30},
			})
		if _, ok := err.(*github.RateLimitError); ok {
			log.Printf("Rate limit error! %#v\n", resp.Rate)
			if resp.Rate.Remaining == 0 {
				fmt.Println("Sleeping until:", time.Until(resp.Rate.Reset.Time))
				time.Sleep(time.Until(resp.Rate.Reset.Time))
			}
			return nil, errRetry
		}
		if err != nil {
			panic(err)
		}
		for _, comment := range comments {
			// NOTE: this can introduce some false-positives if author quotes an earlier "lgtm" comment.
			if strings.Contains(comment.GetBody(), ":lgtm:") || strings.Contains(comment.GetBody(), "LGTM") {
				l := fmt.Sprintln(
					resp.Rate.Remaining, pr.GetBase().GetRepo().GetName(), pr.GetNumber(),
					pr.GetUser().GetLogin(), comment.GetUser().GetLogin(), pr.GetMergedAt().Format(time.RFC3339),
				)
				if pr.GetUser().GetLogin() != comment.GetUser().GetLogin() {
					users[comment.GetUser().GetLogin()] = strings.TrimSpace(l)
				}
			}
		}
		for _, v := range users {
			prLines = append(prLines, v)
		}
	}
	if len(reviews) == 0 && len(pr.RequestedReviewers) > 0 {
		for _, user := range pr.RequestedReviewers {
			l := fmt.Sprintln(
				resp.Rate.Remaining, pr.GetBase().GetRepo().GetName(), pr.GetNumber(),
				pr.GetUser().GetLogin(), user.GetLogin(), pr.GetMergedAt().Format(time.RFC3339),
			)
			prLines = append(prLines, strings.TrimSpace(l))
		}
	}
	if len(reviews) > 0 {
		users := map[string]string{}
		for _, review := range reviews {
			// NOTE: it's possible for users to approve a PR multiple times.
			// Collect one per user.
			l := fmt.Sprintln(
				resp.Rate.Remaining, pr.GetBase().GetRepo().GetName(), pr.GetNumber(),
				pr.GetUser().GetLogin(), review.GetUser().GetLogin(), pr.GetMergedAt().Format(time.RFC3339),
			)
			users[review.GetUser().GetLogin()] = strings.TrimSpace(l)
		}
		for _, v := range users {
			prLines = append(prLines, v)
		}
	}
	return prLines, nil
}

//
func checkAllPRs(ctx context.Context, client *github.Client, repo string) error {
	opt2 := &github.PullRequestListOptions{
		State:       "closed",
		ListOptions: github.ListOptions{PerPage: 50},
	}
	prLines := []string{}
	for {
		// values, resp, err := run
		prs, prResp, err := client.PullRequests.List(ctx, fGithubOwner, repo, opt2)
		if _, ok := err.(*github.RateLimitError); ok {
			log.Printf("rate limit error! %#v\n", prResp.Rate)
			if prResp.Rate.Remaining == 0 {
				fmt.Println("Sleeping until:", time.Until(prResp.Rate.Reset.Time))
				time.Sleep(time.Until(prResp.Rate.Reset.Time))
			}
			// retry if rate limit error
			continue
		}
		// return on error
		if err != nil {
			return err
		}
		// run function on each value
		for i := 0; i < len(prs); {
			lines, err := checkPR(ctx, client, prs[i])
			if err == errRetry {
				continue
			}
			if err != nil {
				return err
			}
			prLines = append(prLines, lines...)
			i++
		}
		// iterate next page.
		if prResp.NextPage == 0 {
			break
		}
		opt2.Page = prResp.NextPage
	}
	writeReposFile("results/"+repo+".txt", prLines)
	return nil
}

func main() {
	flag.Parse()
	if fAuthtoken == "" || fGithubOwner == "" {
		flag.Usage()
		os.Exit(1)
	}

	client := NewClient(fAuthtoken)
	ctx := context.Background()
	repoNames := getRepos(client)

	fmt.Println("Getting PRs")
	if fGithubRepo != "" {
		repoNames = []string{fGithubRepo}
	}

	for _, repo := range repoNames {
		fmt.Println(repo)
		err := checkAllPRs(ctx, client, repo)
		if err != nil {
			panic(err)
		}
	}
}
