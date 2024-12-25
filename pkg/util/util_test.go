package util

import (
	"fmt"
	"testing"

	go_test_ "github.com/pefish/go-test"
)

func TestGetGitShortCommitHash(t *testing.T) {
	r, err := GetGitShortCommitHash("/Users/pefish/Work/golang/ci-tool")
	go_test_.Equal(t, nil, err)
	fmt.Printf("_%s_\n", r)
}

func TestContainerExists(t *testing.T) {
	r, err := ContainerExists("redis1")
	go_test_.Equal(t, nil, err)
	fmt.Println(r)
}

func TestListProjectContainers(t *testing.T) {
	r, err := ListProjectContainers("fsgsf")
	go_test_.Equal(t, nil, err)
	for _, a := range r {
		fmt.Printf("--%s--\n", a)
	}
}
