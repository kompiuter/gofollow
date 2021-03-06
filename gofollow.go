/*
MIT License

Copyright (c) 2016 Kyriacos Kyriacou

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

/*
Package gofollow provides an automated way to follow new users
on Twitter which are found by searching with the search term provided

Flags:
-s    (search term)
-max  (max number of users to follow)
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/ChimeraCoder/anaconda"
)

// searchTerm is the search term used by the search functions
var searchTerm = flag.String("s", "", "(required) search term to find users by (i.e. gopher)")

// maxFollow is the maximum number of users that the application should follow. It is
// an upper bound since there might be a case where not enough users are found to follow
var maxFollow = flag.Int("max", 50, "(optional) max number of users to follow (hard maximum of 100 to avoid limiting by Twitter)")

// alreadyFollowing is a slice that contains all users that are already being followed.
// This is used so that the application does not attempt to follow users that are already friends
var alreadyFollowing []anaconda.User

// toFollow is a slice that should be populated with the users found by the search functions and
// will subsequently be used to follow each user
var toFollow []anaconda.User

func main() {
	api, err := newTwitterAPI()
	if err != nil {
		log.Fatal(err)
	}

	flag.Parse()
	if len(*searchTerm) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}
	const hardMaxFollow = 100
	if *maxFollow > hardMaxFollow {
		*maxFollow = hardMaxFollow
	}

	alreadyFollowing = getAllFriends(api)

	// Start finding users

	fmt.Println("Finding users...")
	done := make(chan struct{})
	go spinner(done) // feedback for user

	found := 0
	n, err := findUsers(api)
	if err != nil {
		log.Fatal(err)
	}
	found += n
	n, err = findUsersByTweet(api)
	if err != nil {
		log.Fatal(err)
	}
	done <- struct{}{}
	found += n
	fmt.Printf("\rFound %d unique users\n", found)
	if found == 0 {
		fmt.Println("Try a broader search term next time!")
		os.Exit(0)
	}

	// Start following users

	const userURLFormat = "https://twitter.com/%s"
	fmt.Println("Following...")
	newFriends := 0
	for _, user := range toFollow {
		err := followUser(api, user.Id)
		if err != nil {
			log.Print(fmt.Errorf("could not follow %s: %v\n", user.ScreenName, err))
			break // could continue, but error is most likely due to limiting by twitter and will fall through
		}
		fmt.Printf("%-40s%-s\n", user.Name, fmt.Sprintf(userURLFormat, user.ScreenName))
		newFriends++
	}
	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Printf("You are now following %d new users!\n", newFriends)
}

// newTwitterAPI returns a new Twitter API using keys from the environment
func newTwitterAPI() (*anaconda.TwitterApi, error) {
	keys := []string{
		"TWITTER_CONSUMER_KEY",
		"TWITTER_CONSUMER_SECRET",
		"TWITTER_ACCESS_TOKEN",
		"TWITTER_ACCESS_SECRET",
	}
	pairs := make(map[string]string, 4)
	for _, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			return nil, fmt.Errorf("environment variable %q required", k)
		}
		pairs[k] = v
	}

	anaconda.SetConsumerKey(pairs["TWITTER_CONSUMER_KEY"])
	anaconda.SetConsumerSecret(pairs["TWITTER_CONSUMER_SECRET"])
	api := anaconda.NewTwitterApi(pairs["TWITTER_ACCESS_TOKEN"], pairs["TWITTER_ACCESS_SECRET"])
	_, err := api.VerifyCredentials()
	if err != nil {
		return nil, err
	}
	return api, nil
}

// spinner displays a spinner on the std output
func spinner(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
			for _, r := range `-\|/` {
				fmt.Printf("\r%c", r)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}
}

// findUsers uses "searchTerm" to search for tweets using the users/search API
// (https://dev.twitter.com/rest/reference/get/users/search).
// Returns the number of users found
func findUsers(api *anaconda.TwitterApi) (int, error) {
	maxCount := 20
	// don't ask for more than required
	usersRequired := *maxFollow - len(toFollow)
	if maxCount > usersRequired {
		maxCount = usersRequired
	}
	if maxCount == 0 {
		return 0, nil
	}
	page := 0
	values := make(url.Values)
	values.Add("count", strconv.Itoa(maxCount))
	values.Add("include_entities", "false")

	added := 0
	for {
		values.Set("page", strconv.Itoa(page))
		resp, err := api.GetUserSearch(*searchTerm, values)
		if err != nil {
			return 0, fmt.Errorf("findUsers: %v", err)
		}
		for _, user := range resp {
			if !isFollowing(user) {
				if len(toFollow) >= *maxFollow { // check if we reached max number of people to follow
					return added, nil
				}
				toFollow = append(toFollow, user)
				added++
			}
		}
		if len(resp) != maxCount { // there are no more pages available
			break
		}
		if len(toFollow) >= *maxFollow { // got number of followers required
			break
		}
		page++
	}

	return added, nil
}

// findUsersByTweet uses "searchTerm" to search for tweets using the search/tweets API
// (https://dev.twitter.com/rest/reference/get/search/tweets).
// Returns the number of users found
func findUsersByTweet(api *anaconda.TwitterApi) (int, error) {
	values := make(url.Values)
	maxCount := 100
	// don't ask for more than required
	usersRequired := *maxFollow - len(toFollow)
	if maxCount > usersRequired {
		maxCount = usersRequired
	}
	if maxCount == 0 {
		return 0, nil
	}
	values.Add("result_type", "mixed")
	values.Add("count", strconv.Itoa(maxCount))
	values.Add("include_entities", "false")
	values.Add("lang", "en")

	added := 0
	fn := func(resp anaconda.SearchResponse) {
		for _, tweet := range resp.Statuses {
			if !isFollowing(tweet.User) {
				if len(toFollow) >= *maxFollow { // check if we reached max number of people to follow
					return
				}
				toFollow = append(toFollow, tweet.User)
				added++
			}
		}
	}

	resp, err := api.GetSearch(*searchTerm, values)
	for {
		if err != nil {
			return 0, fmt.Errorf("findUsersByTweet: %v", err)
		}
		fn(resp)
		if len(resp.Statuses) != maxCount { // no more pages
			break
		}
		if len(toFollow) >= *maxFollow { // got number of followers required
			break
		}
		resp, err = resp.GetNext(api)
	}

	return added, nil
}

// getAllFriends returns a slice of all friends of the user
func getAllFriends(api *anaconda.TwitterApi) []anaconda.User {
	var friends []anaconda.User
	ch := api.GetFriendsListAll(nil)
	for f := range ch {
		friends = append(friends, f.Friends...)
	}
	return friends
}

// isFollowing returns true if the user passed is already a friend or is
// already scheduled to be followed
func isFollowing(user anaconda.User) bool {
	for _, u := range alreadyFollowing {
		if u.Id == user.Id {
			return true
		}
	}
	for _, u := range toFollow {
		if u.Id == user.Id {
			return true
		}
	}
	return false
}

// followUser uses the "TwitterApi" passed to follow a "userID".
// If followed without error, the user's screen name is returned.
// Attempting to follow a user which is already a friend will treat
// it as a user who is not a friend
func followUser(api *anaconda.TwitterApi, userID int64) error {
	_, err := api.FollowUserId(userID, nil)
	if err != nil {
		return fmt.Errorf("follow %d: %v\n", userID, err)
	}
	return nil
}
