package checkers

import (
	"strings"
	"testing"
)

// workflow mixes pinned, unpinned, local and docker references.
const workflow = `name: ci
on: push
jobs:
  a:
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      - uses: actions/setup-go@v6
      - name: local
        uses: ./.github/actions/x
      - uses: "docker://alpine:3"
      - uses: docker://alpine@sha256:abc
      - uses: some/action
  b:
    uses: org/repo/.github/workflows/x.yml@main
`

func TestFindUsesAndPinned(t *testing.T) {
	t.Parallel()
	refs := FindUses([]byte(workflow))
	if len(refs) != 7 || refs[0].Line != 6 || refs[2].Ref != "./.github/actions/x" {
		t.Fatalf("refs = %+v", refs)
	}
	is := checkActionsPinned(nil, "ci.yml", []byte(workflow))
	var got []string
	for _, i := range is {
		got = append(got, i.Message)
	}
	want := "actions/setup-go@v6 is not pinned to a commit SHA|docker://alpine:3 is not pinned to a commit SHA|some/action is not pinned to a commit SHA|org/repo/.github/workflows/x.yml@main is not pinned to a commit SHA"
	if strings.Join(got, "|") != want {
		t.Fatalf("got %s", strings.Join(got, "|"))
	}
}

func TestWorkflowPermissions(t *testing.T) {
	t.Parallel()
	tests := map[string]int{
		workflow:                      1,
		"permissions: {}\non: push\n": 0,
		"on: push\npermissions:\n  contents: read\n": 0,
		"- just\n- a list\n":                         0,
		"{{ not yaml":                                0,
		"":                                           0,
	}
	for in, want := range tests {
		if got := len(checkWorkflowPermissions(nil, "ci.yml", []byte(in))); got != want {
			t.Errorf("%q: %d issues, want %d", in, got, want)
		}
	}
}

// FuzzPinned checks that only immutable references count as pinned.
func FuzzPinned(f *testing.F) {
	f.Add("actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1")
	f.Add("actions/checkout@v4")
	f.Add("./local")
	f.Add("docker://x@sha256:1")
	f.Fuzz(func(t *testing.T, ref string) {
		if !Pinned(ref) {
			return
		}
		_, version, _ := strings.Cut(ref, "@")
		ok := strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "docker://") && strings.Contains(ref, "@sha256:") || commitSHA.MatchString(version)
		if !ok {
			t.Fatalf("%q counted as pinned", ref)
		}
	})
}
