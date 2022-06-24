package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/sets"
	github "k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
	"k8s.io/test-infra/prow/tide"

	"github.com/openshift/ci-tools/pkg/testhelper"
)

type MyFakeClient struct {
	*fakegithub.FakeClient
}

func (f *MyFakeClient) QueryWithGitHubAppsSupport(ctx context.Context, q interface{}, vars map[string]interface{}, org string) error {
	return nil
}

func (f *MyFakeClient) GetRef(owner, repo, ref string) (string, error) {
	if owner == "failed test" {
		return "", fmt.Errorf("failed")
	}
	return "abcde", nil
}

func TestLoadConfig(t *testing.T) {
	c := &Config{
		Retester: Retester{
			map[string]Oranization{"openshift": {
				MaxRetestsForShaAndBase: 3, MaxRetestsForSha: 9, Repos: map[string]Repo{"ci-tools": {MaxRetestsForShaAndBase: 3, MaxRetestsForSha: 8, Enabled: true}},
			},
				"no-openshift": {
					MaxRetestsForShaAndBase: 1, MaxRetestsForSha: 1, Repos: map[string]Repo{"test": {MaxRetestsForShaAndBase: 3, MaxRetestsForSha: 7, Enabled: true},
						"disabled-repo": {MaxRetestsForShaAndBase: 2, MaxRetestsForSha: 4}},
				}},
		}}

	testCases := []struct {
		name          string
		file          string
		expected      *Config
		expectedError error
	}{
		{
			name:     "basic case",
			file:     "testdata/testconfig/config.yaml",
			expected: c,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := loadConfig(tc.file)
			if diff := cmp.Diff(tc.expectedError, err, testhelper.EquateErrorMessage); diff != "" {
				t.Errorf("Error differs from expected:\n%s", diff)
			}
			if tc.expectedError == nil {
				if diff := cmp.Diff(tc.expected, actual); diff != "" {
					t.Errorf("%s differs from expected:\n%s", tc.name, diff)
				}
			}
		})
	}
}

func TestGetMaxRetests(t *testing.T) {
	c := &Config{
		Retester: Retester{
			map[string]Oranization{"openshift": {
				MaxRetestsForShaAndBase: 6, MaxRetestsForSha: 15, Repos: map[string]Repo{"ci-tools": {MaxRetestsForShaAndBase: 3, MaxRetestsForSha: 8, Enabled: true},
					"repo": {MaxRetestsForShaAndBase: 3, MaxRetestsForSha: 8, Enabled: false}},
			}},
		}}
	var configuredRepo githubv4.String = "ci-tools"
	var nonConfiguredRepo githubv4.String = "repo"
	var configuredOrg githubv4.String = "openshift"
	var nonconfiguredOrg githubv4.String = "org"
	var num githubv4.Int = 123
	testCases := []struct {
		name     string
		pr       tide.PullRequest
		config   *Config
		expected MaxRetests
	}{
		{
			name: "configured org and non-configured repo",
			pr: tide.PullRequest{
				Number: num,
				Author: struct{ Login githubv4.String }{Login: configuredOrg},
				Repository: struct {
					Name          githubv4.String
					NameWithOwner githubv4.String
					Owner         struct{ Login githubv4.String }
				}{Name: nonConfiguredRepo, Owner: struct{ Login githubv4.String }{Login: configuredOrg}},
			},
			config:   c,
			expected: MaxRetests{15, 6},
		},
		{
			name: "configured org and configured repo",
			pr: tide.PullRequest{
				Number: num,
				Author: struct{ Login githubv4.String }{Login: configuredOrg},
				Repository: struct {
					Name          githubv4.String
					NameWithOwner githubv4.String
					Owner         struct{ Login githubv4.String }
				}{Name: configuredRepo, Owner: struct{ Login githubv4.String }{Login: configuredOrg}},
			},
			config:   c,
			expected: MaxRetests{8, 3},
		},
		{
			name: "non-configured org and non-configured repo",
			pr: tide.PullRequest{
				Number: num,
				Author: struct{ Login githubv4.String }{Login: nonconfiguredOrg},
				Repository: struct {
					Name          githubv4.String
					NameWithOwner githubv4.String
					Owner         struct{ Login githubv4.String }
				}{Name: nonConfiguredRepo, Owner: struct{ Login githubv4.String }{Login: nonconfiguredOrg}},
			},
			config:   c,
			expected: MaxRetests{9, 3},
		},
		{
			name: "non-configured org and configured repo",
			pr: tide.PullRequest{
				Number: num,
				Author: struct{ Login githubv4.String }{Login: nonconfiguredOrg},
				Repository: struct {
					Name          githubv4.String
					NameWithOwner githubv4.String
					Owner         struct{ Login githubv4.String }
				}{Name: configuredRepo, Owner: struct{ Login githubv4.String }{Login: nonconfiguredOrg}},
			},
			config:   c,
			expected: MaxRetests{9, 3},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.config.getMaxRetests(tc.pr)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("%s differs from expected:\n%s", tc.name, diff)
			}
		})
	}
}

func TestUpdateEnabledOrgAndRepo(t *testing.T) {
	c := &Config{
		Retester: Retester{
			map[string]Oranization{"openshift": {
				MaxRetestsForShaAndBase: 3, MaxRetestsForSha: 9, Repos: map[string]Repo{"ci-tools": {MaxRetestsForShaAndBase: 3, MaxRetestsForSha: 8, Enabled: true}},
			},
				"no-openshift": {
					MaxRetestsForShaAndBase: 1, MaxRetestsForSha: 1, Repos: map[string]Repo{"test": {MaxRetestsForShaAndBase: 3, MaxRetestsForSha: 7, Enabled: true},
						"disabled-repo": {MaxRetestsForShaAndBase: 2, MaxRetestsForSha: 4}},
				}},
		}}
	testCases := []struct {
		name          string
		config        *Config
		repos         sets.String
		orgs          sets.String
		expectedRepos sets.String
		expectedOrgs  sets.String
	}{
		{
			name:          "basic case",
			config:        c,
			repos:         sets.NewString("openshift/test"),
			orgs:          sets.NewString("openshift"),
			expectedRepos: sets.NewString("openshift/test", "openshift/ci-tools", "no-openshift/test"),
			expectedOrgs:  sets.NewString("openshift"),
		},
		{
			name:          "no repo and no org from arguments",
			config:        c,
			repos:         sets.NewString(),
			orgs:          sets.NewString(),
			expectedRepos: sets.NewString("openshift/ci-tools", "no-openshift/test"),
			expectedOrgs:  sets.NewString(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.updateEnabledRepos(&tc.repos)
			if diff := cmp.Diff(tc.expectedOrgs, tc.orgs); diff != "" {
				t.Errorf("%s differs from expectedOrgs:\n%s", tc.name, diff)
			}
			if diff := cmp.Diff(tc.expectedRepos, tc.repos); diff != "" {
				t.Errorf("%s differs from expectedRepos:\n%s", tc.name, diff)
			}
		})
	}
}

func TestRetestOrBackoff(t *testing.T) {
	ghc := &MyFakeClient{fakegithub.NewFakeClient()}
	var name githubv4.String = "repo"
	var owner githubv4.String = "org"
	var fail githubv4.String = "failed test"
	var num githubv4.Int = 123
	var num2 githubv4.Int = 321
	pr123 := github.PullRequest{}
	pr321 := github.PullRequest{}
	ghc.PullRequests = map[int]*github.PullRequest{123: &pr123, 321: &pr321}
	logger := logrus.NewEntry(logrus.StandardLogger())

	testCases := []struct {
		name          string
		pr            tide.PullRequest
		c             *retestController
		expected      string
		expectedError error
	}{
		{
			name: "basic case",
			pr: tide.PullRequest{
				Number: num,
				Author: struct{ Login githubv4.String }{Login: owner},
				Repository: struct {
					Name          githubv4.String
					NameWithOwner githubv4.String
					Owner         struct{ Login githubv4.String }
				}{Name: name, Owner: struct{ Login githubv4.String }{Login: owner}},
			},
			c: &retestController{
				ghClient: ghc,
				logger:   logger,
				backoff:  &backoffCache{cache: map[string]*PullRequest{}, logger: logger},
			},
			expected: "/retest-required\n\nRemaining retests: 2 against base HEAD abcde and 8 for PR HEAD  in total\n",
		},
		{
			name: "failed test",
			pr: tide.PullRequest{
				Number: num2,
				Author: struct{ Login githubv4.String }{Login: fail},
				Repository: struct {
					Name          githubv4.String
					NameWithOwner githubv4.String
					Owner         struct{ Login githubv4.String }
				}{Name: name, Owner: struct{ Login githubv4.String }{Login: fail}},
			},
			c: &retestController{
				ghClient: ghc,
				logger:   logger,
				backoff:  &backoffCache{cache: map[string]*PullRequest{}, logger: logger},
			},
			expected:      "",
			expectedError: fmt.Errorf("failed"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.c.retestOrBackoff(tc.pr)
			if diff := cmp.Diff(tc.expectedError, err, testhelper.EquateErrorMessage); diff != "" {
				t.Errorf("Error differs from expected:\n%s", diff)
			}
			if tc.expectedError == nil {
				actual := ""
				if len(ghc.IssueComments[int(tc.pr.Number)]) != 0 {
					actual = ghc.IssueComments[int(tc.pr.Number)][0].Body
				}
				if diff := cmp.Diff(tc.expected, actual); diff != "" {
					t.Errorf("%s differs from expected:\n%s", tc.name, diff)
				}
			}
		})
	}
}

func TestEnabledPRs(t *testing.T) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	testCases := []struct {
		name       string
		c          *retestController
		candidates map[string]tide.PullRequest
		expected   map[string]tide.PullRequest
	}{
		{
			name: "basic case",
			c: &retestController{
				enableOnRepos: sets.NewString("openshift/ci-tools"),
				enableOnOrgs:  sets.NewString("org-a"),
				logger:        logger,
			},
			candidates: map[string]tide.PullRequest{
				"a": {
					Number: 1,
					Repository: struct {
						Name          githubv4.String
						NameWithOwner githubv4.String
						Owner         struct{ Login githubv4.String }
					}{Name: "ci-tools", Owner: struct{ Login githubv4.String }{Login: "openshift"}},
				},
				"b": {
					Number: 1,
					Repository: struct {
						Name          githubv4.String
						NameWithOwner githubv4.String
						Owner         struct{ Login githubv4.String }
					}{Name: "some-tools", Owner: struct{ Login githubv4.String }{Login: "openshift"}},
				},
				"c": {
					Number: 1,
					Repository: struct {
						Name          githubv4.String
						NameWithOwner githubv4.String
						Owner         struct{ Login githubv4.String }
					}{Name: "some-tools", Owner: struct{ Login githubv4.String }{Login: "org-a"}},
				},
				"d": {
					Number: 1,
					Repository: struct {
						Name          githubv4.String
						NameWithOwner githubv4.String
						Owner         struct{ Login githubv4.String }
					}{Name: "some-tools", Owner: struct{ Login githubv4.String }{Login: "org-b"}},
				},
			},
			expected: map[string]tide.PullRequest{
				"a": {
					Number: 1,
					Repository: struct {
						Name          githubv4.String
						NameWithOwner githubv4.String
						Owner         struct{ Login githubv4.String }
					}{Name: "ci-tools", Owner: struct{ Login githubv4.String }{Login: "openshift"}},
				},
				"c": {
					Number: 1,
					Repository: struct {
						Name          githubv4.String
						NameWithOwner githubv4.String
						Owner         struct{ Login githubv4.String }
					}{Name: "some-tools", Owner: struct{ Login githubv4.String }{Login: "org-a"}},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.c.enabledPRs(tc.candidates)
			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("%s differs from expected:\n%s", tc.name, diff)
			}
		})
	}
}
