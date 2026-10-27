package checkers

import (
	"bufio"
	"bytes"
	"io/fs"
	"strings"

	"github.com/rossedman/canon/internal/findings"
)

// Stage is one FROM block of a Dockerfile.
type Stage struct {
	Image string
	Name  string
	Line  int
	Users []User
}

// User is a USER instruction.
type User struct {
	Value string
	Line  int
}

// ParseDockerfile splits a Dockerfile into stages, joining continuation lines
// and skipping comments. Only FROM and USER are kept; nothing else matters to
// canon's checks.
func ParseDockerfile(data []byte) []Stage {
	var stages []Stage
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo, start := 0, 0
	var logical strings.Builder
	for sc.Scan() {
		lineNo++
		line := strings.TrimRight(sc.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if logical.Len() == 0 {
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			start = lineNo
		} else if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if cont, ok := strings.CutSuffix(trimmed, `\`); ok {
			logical.WriteString(cont)
			logical.WriteByte(' ')
			continue
		}
		logical.WriteString(trimmed)
		stages = addInstruction(stages, logical.String(), start)
		logical.Reset()
	}
	if logical.Len() > 0 {
		stages = addInstruction(stages, logical.String(), start)
	}
	return stages
}

// addInstruction applies one logical instruction to the stage list.
func addInstruction(stages []Stage, inst string, line int) []Stage {
	fields := strings.Fields(inst)
	if len(fields) == 0 {
		return stages
	}
	switch strings.ToUpper(fields[0]) {
	case "FROM":
		args := fields[1:]
		for len(args) > 0 && strings.HasPrefix(args[0], "--") {
			args = args[1:]
		}
		st := Stage{Line: line}
		if len(args) > 0 {
			st.Image = args[0]
		}
		if len(args) >= 3 && strings.EqualFold(args[1], "AS") {
			st.Name = args[2]
		}
		stages = append(stages, st)
	case "USER":
		if len(stages) > 0 && len(fields) > 1 {
			last := &stages[len(stages)-1]
			last.Users = append(last.Users, User{Value: fields[1], Line: line})
		}
	}
	return stages
}

// stageByName finds an earlier stage by its AS name.
func stageByName(stages []Stage, name string, before int) (int, bool) {
	for i := before - 1; i >= 0; i-- {
		if stages[i].Name != "" && strings.EqualFold(stages[i].Name, name) {
			return i, true
		}
	}
	return 0, false
}

// FinalBase follows the last stage through any stage aliases to the external
// image it is built on, and returns that image with every USER along the way.
func FinalBase(stages []Stage) (image string, users []User, ok bool) {
	if len(stages) == 0 {
		return "", nil, false
	}
	i := len(stages) - 1
	var chain []int
	for range len(stages) {
		chain = append(chain, i)
		j, isStage := stageByName(stages, stages[i].Image, i)
		if !isStage {
			break
		}
		i = j
	}
	for k := len(chain) - 1; k >= 0; k-- {
		users = append(users, stages[chain[k]].Users...)
	}
	return stages[i].Image, users, true
}

// minimalBases are runtime image prefixes with no shell or package manager.
var minimalBases = []string{"scratch", "gcr.io/distroless/", "cgr.dev/chainguard/"}

// isMinimal reports whether image is a distroless, chainguard or scratch base.
func isMinimal(image string) bool {
	img := strings.ToLower(image)
	for _, p := range minimalBases {
		if img == p || strings.HasPrefix(img, p) {
			return true
		}
	}
	return false
}

// checkFinalStageBase reports a final stage built on a full OS image.
func checkFinalStageBase(_ fs.FS, _ string, data []byte) []Issue {
	stages := ParseDockerfile(data)
	image, _, ok := FinalBase(stages)
	if !ok || image == "" || strings.Contains(image, "$") || isMinimal(image) {
		return nil
	}
	last := stages[len(stages)-1]
	return []Issue{{
		Line:    last.Line,
		Message: "Final stage uses " + image,
		Fix:     &findings.Fix{Kind: "replace", Suggestion: "FROM gcr.io/distroless/static-debian12:nonroot"},
	}}
}

// checkBaseDigest reports every external FROM without a sha256 digest.
func checkBaseDigest(_ fs.FS, _ string, data []byte) []Issue {
	stages := ParseDockerfile(data)
	var out []Issue
	for i, st := range stages {
		img := st.Image
		if img == "" || strings.EqualFold(img, "scratch") || strings.Contains(img, "$") || strings.Contains(img, "@sha256:") {
			continue
		}
		if _, isStage := stageByName(stages, img, i); isStage {
			continue
		}
		out = append(out, Issue{
			Line:    st.Line,
			Message: "FROM " + img + " is not pinned to a digest",
			Fix:     &findings.Fix{Kind: "pin", Suggestion: "FROM " + img + "@sha256:<digest from docker buildx imagetools inspect " + img + ">"},
		})
	}
	return out
}

// isRoot reports whether a USER value is the root user.
func isRoot(u string) bool {
	name, _, _ := strings.Cut(u, ":")
	return name == "root" || name == "0"
}

// checkFinalStageUser reports a final stage that runs as root, either
// explicitly or because no USER is set on a base that defaults to root.
func checkFinalStageUser(_ fs.FS, _ string, data []byte) []Issue {
	stages := ParseDockerfile(data)
	image, users, ok := FinalBase(stages)
	if !ok || strings.Contains(image, "$") {
		return nil
	}
	fix := &findings.Fix{Kind: "insert", Suggestion: "USER 65532:65532"}
	if len(users) > 0 {
		u := users[len(users)-1]
		if isRoot(u.Value) {
			return []Issue{{Line: u.Line, Message: "Final stage runs as root (USER " + u.Value + ")", Fix: fix}}
		}
		return nil
	}
	img := strings.ToLower(image)
	if strings.Contains(img, "nonroot") || strings.HasPrefix(img, "cgr.dev/chainguard/") {
		return nil
	}
	last := stages[len(stages)-1]
	return []Issue{{Line: last.Line, Message: "Final stage sets no USER, so it runs as root", Fix: fix}}
}
