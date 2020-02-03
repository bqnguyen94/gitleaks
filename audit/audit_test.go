package audit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"sort"
	"testing"

	"github.com/zricethezav/gitleaks/v3/config"
	"github.com/zricethezav/gitleaks/v3/manager"
	"github.com/zricethezav/gitleaks/v3/options"

	"github.com/sergi/go-diff/diffmatchpatch"
)

const testRepoBase = "../test_data/test_repos/"

func TestAudit(t *testing.T) {
	moveDotGit("dotGit", ".git")
	defer moveDotGit(".git", "dotGit")
	tests := []struct {
		description string
		opts        options.Options
		wantPath    string
		wantErr     error
		emptyRepo   bool
		wantEmpty   bool
	}{
		{
			description: "test local repo one aws leak",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_local_repo_one_aws_leak.json.got",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_repo_one_aws_leak.json",
		},
		{
			description: "test local repo one aws leak threaded",
			opts: options.Options{
				Threads:      runtime.GOMAXPROCS(0),
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_local_repo_one_aws_leak.json.got",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_repo_one_aws_leak.json",
		},
		{
			description: "test non existent repo",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/no_repo_here",
				ReportFormat: "json",
			},
			emptyRepo: true,
		},
		{
			description: "test local repo one aws leak whitelisted",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				ReportFormat: "json",
				Config:       "../test_data/test_configs/aws_key_whitelist_python_files.toml",
			},
			wantEmpty: true,
		},
		{
			description: "test local repo two leaks",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_2",
				Report:       "../test_data/test_local_repo_two_leaks.json.got",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_repo_two_leaks.json",
		},
		{
			description: "test local repo two leaks from commit",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_2",
				Report:       "../test_data/test_local_repo_two_leaks_commit_from.json.got",
				ReportFormat: "json",
				CommitFrom:   "996865bb912f3bc45898a370a13aadb315014b55",
			},
			wantPath: "../test_data/test_local_repo_two_leaks_commit_from.json",
		},
		{
			description: "test local repo two leaks to commit",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_2",
				Report:       "../test_data/test_local_repo_two_leaks_commit_to.json.got",
				ReportFormat: "json",
				CommitTo:     "996865bb912f3bc45898a370a13aadb315014b55",
			},
			wantPath: "../test_data/test_local_repo_two_leaks_commit_to.json",
		},
		{
			description: "test local repo two leaks range commit",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_2",
				Report:       "../test_data/test_local_repo_two_leaks_commit_range.json.got",
				ReportFormat: "json",
				CommitFrom:   "d8ac0b73aeeb45843319cdc5ce506516eb49bf7a",
				CommitTo:     "51f6dcf6b89b93f4075ba92c400b075631a6cc93",
			},
			wantPath: "../test_data/test_local_repo_two_leaks_commit_range.json",
		},
		{
			description: "test local repo two leaks globally whitelisted",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_2",
				Config:       "../test_data/test_configs/aws_key_global_whitelist_file.toml",
				ReportFormat: "json",
			},
			wantEmpty: true,
		},
		{
			description: "test local repo two leaks whitelisted",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_2",
				Config:       "../test_data/test_configs/aws_key_whitelist_files.toml",
				ReportFormat: "json",
			},
			wantEmpty: true,
		},
		{
			description: "test local repo three leaks dev branch",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_3",
				Report:       "../test_data/test_local_repo_three_leaks.json.got",
				Config:       "../test_data/test_configs/aws_key.toml",
				Branch:       "dev",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_repo_three_leaks.json",
		},
		{
			description: "test local repo branch does not exist",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_3",
				Branch:       "nobranch",
				ReportFormat: "json",
			},
			wantEmpty: true,
		},
		{
			description: "test local repo one aws leak single commit",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_local_repo_one_aws_leak_commit.json.got",
				Commit:       "6557c92612d3b35979bd426d429255b3bf9fab74",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_repo_one_aws_leak_commit.json",
		},
		{
			description: "test local repo one aws leak AND leak on python files",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_local_repo_one_aws_leak_and_file_leak.json.got",
				Config:       "../test_data/test_configs/aws_key_file_regex.toml",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_repo_one_aws_leak_and_file_leak.json",
		},
		{
			description: "test owner path",
			opts: options.Options{
				OwnerPath:    "../test_data/test_repos/",
				Report:       "../test_data/test_local_owner_aws_leak.json.got",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_owner_aws_leak.json",
		},
		{
			description: "test entropy",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_entropy.json.got",
				Config:       "../test_data/test_configs/entropy.toml",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_entropy.json",
		},
		{
			description: "test entropy and regex",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_regex_entropy.json.got",
				Config:       "../test_data/test_configs/regex_entropy.toml",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_regex_entropy.json",
		},
		{
			description: "test local repo four entropy alternative config",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_4",
				Report:       "../test_data/test_local_repo_four_alt_config_entropy.json.got",
				RepoConfig:   true,
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_repo_four_alt_config_entropy.json",
		},
		{
			description: "test local repo four entropy alternative config",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_regex_whitelist.json.got",
				Config:       "../test_data/test_configs/aws_key_aws_whitelisted.toml",
				ReportFormat: "json",
			},
			wantEmpty: true,
		},
		{
			description: "test local repo one aws leak timeout",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_local_repo_one_aws_leak.json.got",
				ReportFormat: "json",
				Timeout:      "10ns",
			},
			wantEmpty: true,
		},
		{
			description: "test local repo one aws leak long timeout",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_local_repo_one_aws_leak.json.got",
				ReportFormat: "json",
				Timeout:      "2m",
			},
			wantPath: "../test_data/test_local_repo_one_aws_leak.json",
		},
		{
			description: "test owner path depth=2",
			opts: options.Options{
				OwnerPath:    "../test_data/test_repos/",
				Report:       "../test_data/test_local_owner_aws_leak_depth_2.json.got",
				ReportFormat: "json",
				Depth:        2,
			},
			wantPath: "../test_data/test_local_owner_aws_leak_depth_2.json",
		},
		{
			description: "test local repo five files at commit",
			opts: options.Options{
				RepoPath:      "../test_data/test_repos/test_repo_5",
				Report:        "../test_data/test_local_repo_five_files_at_commit.json.got",
				FilesAtCommit: "a4c9fb737d5552fd96fce5cc7eedb23353ba9ed0",
				ReportFormat:  "json",
			},
			wantPath: "../test_data/test_local_repo_five_files_at_commit.json",
		},
		{
			description: "test local repo five at commit",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_5",
				Report:       "../test_data/test_local_repo_five_commit.json.got",
				Commit:       "a4c9fb737d5552fd96fce5cc7eedb23353ba9ed0",
				ReportFormat: "json",
			},
			wantPath: "../test_data/test_local_repo_five_commit.json",
		},
	}

	for _, test := range tests {
		fmt.Println(test.description)
		cfg, err := config.NewConfig(test.opts)
		if err != nil {
			t.Error(err)
		}

		m, err := manager.NewManager(test.opts, cfg)
		if err != nil {
			t.Error(err)
		}

		err = Run(m)

		if test.wantErr != nil {
			if err == nil {
				t.Errorf("did not receive wantErr: %v", test.wantErr)
			}
			if err.Error() != test.wantErr.Error() {
				t.Errorf("wantErr does not equal err received: %v", err.Error())
			}
			continue
		}

		err = m.Report()

		if test.wantEmpty {
			if len(m.GetLeaks()) != 0 {
				t.Errorf("wanted no leaks but got some instead: %+v", m.GetLeaks())
			}
			continue
		}

		if test.wantPath != "" {
			err := fileCheck(test.wantPath, test.opts.Report)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func TestAuditUncommited(t *testing.T) {
	moveDotGit("dotGit", ".git")
	defer moveDotGit(".git", "dotGit")
	tests := []struct {
		description  string
		opts         options.Options
		wantPath     string
		wantErr      error
		emptyRepo    bool
		wantEmpty    bool
		fileToChange string
		addition     string
	}{
		{
			description: "test audit local one leak",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Report:       "../test_data/test_local_repo_one_aws_leak_uncommitted.json.got",
				Uncommited:   true,
				ReportFormat: "json",
			},
			wantPath:     "../test_data/test_local_repo_one_aws_leak_uncommitted.json",
			fileToChange: "server.test.py",
			addition:     " aws_access_key_id='AKIAIO5FODNN7DXAMPLE'\n\n",
		},
		{
			description: "test audit local no leak",
			opts: options.Options{
				RepoPath:     "../test_data/test_repos/test_repo_1",
				Uncommited:   true,
				ReportFormat: "json",
			},
			wantEmpty:    true,
			fileToChange: "server.test.py",
			addition:     "nothing bad",
		},
	}
	for _, test := range tests {
		fmt.Println(test.description)
		old, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", test.opts.RepoPath, test.fileToChange))
		if err != nil {
			t.Error(err)
		}
		altered, err := os.OpenFile(fmt.Sprintf("%s/%s", test.opts.RepoPath, test.fileToChange),
			os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			t.Error(err)
		}

		_, err = altered.WriteString(test.addition)
		if err != nil {
			t.Error(err)
		}

		cfg, err := config.NewConfig(test.opts)
		if err != nil {
			t.Error(err)
		}
		m, err := manager.NewManager(test.opts, cfg)
		if err != nil {
			t.Error(err)
		}

		if err := Run(m); err != nil {
			t.Error(err)
		}

		if err := m.Report(); err != nil {
			t.Error(err)
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", test.opts.RepoPath, test.fileToChange), old, 0)
		if err != nil {
			t.Error(err)
		}

		if test.wantEmpty {
			continue
		}

		if test.wantPath != "" {
			err := fileCheck(test.wantPath, test.opts.Report)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func fileCheck(wantPath, gotPath string) error {
	var (
		gotLeaks  []manager.Leak
		wantLeaks []manager.Leak
	)
	want, err := ioutil.ReadFile(wantPath)
	if err != nil {
		return err
	}

	got, err := ioutil.ReadFile(gotPath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(got, &gotLeaks)
	if err != nil {
		return err
	}

	err = json.Unmarshal(want, &wantLeaks)
	if err != nil {
		return nil
	}

	sort.Slice(gotLeaks, func(i, j int) bool { return (gotLeaks)[i].Commit < (gotLeaks)[j].Commit })
	sort.Slice(wantLeaks, func(i, j int) bool { return (wantLeaks)[i].Commit < (wantLeaks)[j].Commit })

	if !reflect.DeepEqual(gotLeaks, wantLeaks) {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(want), string(got), false)
		return fmt.Errorf("does not equal: %s", dmp.DiffPrettyText(diffs))
	}
	if err := os.Remove(gotPath); err != nil {
		return err
	}
	return nil
}

func moveDotGit(from, to string) error {
	repoDirs, err := ioutil.ReadDir("../test_data/test_repos")
	if err != nil {
		return err
	}
	for _, dir := range repoDirs {
		if !dir.IsDir() {
			continue
		}
		err = os.Rename(fmt.Sprintf("%s/%s/%s", testRepoBase, dir.Name(), from),
			fmt.Sprintf("%s/%s/%s", testRepoBase, dir.Name(), to))
		if err != nil {
			return err
		}
	}
	return nil
}
